// SPDX-License-Identifier: MIT

package artifactory

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	rtauth "github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/config"

	"github.com/tenthirtyam/artifactory-content-library/internal/auth"
	"github.com/tenthirtyam/artifactory-content-library/internal/logging"
	"github.com/tenthirtyam/artifactory-content-library/internal/ratelimit"
	"github.com/tenthirtyam/artifactory-content-library/internal/security"
)

// ChildItem is a directory listing entry.
type ChildItem struct {
	URI    string `json:"uri"`
	Folder bool   `json:"folder"`
}

// FileMeta holds storage metadata for a file.
type FileMeta struct {
	Size         int64
	SHA1         string
	MD5          string
	LastModified string
}

// StorageClient is the Artifactory operations used by the generator.
type StorageClient interface {
	FileExists(ctx context.Context, relPath string) (bool, error)
	GetFileInfo(ctx context.Context, relPath string) (*FileMeta, error)
	ListItems(ctx context.Context, relPath string) ([]ChildItem, error)
	Download(ctx context.Context, relPath string) ([]byte, error)
	Upload(ctx context.Context, relPath string, content []byte, contentType string) error
	Delete(ctx context.Context, relPath string) error
	Repo() string
	BaseURL() string
}

// Client wraps jfrog-client-go with rate limiting and retries.
type Client struct {
	manager    artifactory.ArtifactoryServicesManager
	baseURL    string
	repo       string
	httpClient *http.Client
	limiter    *ratelimit.Limiter
	creds      *auth.Credentials
	retry      *retryLogic
}

// NewClient creates an Artifactory client using jfrog-client-go.
func NewClient(creds *auth.Credentials) (*Client, error) {
	baseURL, err := security.ValidateArtifactoryURL(creds.URL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(creds.Repo) == "" {
		return nil, fmt.Errorf("repository name cannot be empty")
	}

	rtDetails := rtauth.NewArtifactoryDetails()
	rtDetails.SetUrl(baseURL + "/")

	switch creds.Method {
	case auth.MethodAPIKey:
		if creds.APIKey == "" {
			return nil, fmt.Errorf("API key is required for api_key authentication")
		}
		rtDetails.SetApiKey(creds.APIKey)
		logging.Audit("Artifactory client authentication configured.",
			"auth_method", string(creds.Method),
			"masked_credential", security.MaskSensitive(creds.APIKey, 4),
			"repo_name", creds.Repo,
		)
	case auth.MethodBasic:
		if creds.Username == "" || creds.Password == "" {
			return nil, fmt.Errorf("username and password are required for basic authentication")
		}
		rtDetails.SetUser(creds.Username)
		rtDetails.SetPassword(creds.Password)
		logging.Audit("Artifactory client authentication configured.",
			"auth_method", string(creds.Method),
			"masked_credential", creds.Username+":"+security.MaskSensitive(creds.Password, 4),
			"repo_name", creds.Repo,
		)
	case auth.MethodToken:
		if creds.Token == "" {
			return nil, fmt.Errorf("token is required for token authentication")
		}
		rtDetails.SetAccessToken(creds.Token)
		logging.Audit("Artifactory client authentication configured.",
			"auth_method", string(creds.Method),
			"masked_credential", security.MaskSensitive(creds.Token, 4),
			"repo_name", creds.Repo,
		)
	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", creds.Method)
	}

	timeout := time.Duration(creds.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(auth.DefaultTimeoutSeconds) * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: !creds.SSLVerify} //nolint:gosec
	httpClient := &http.Client{Timeout: timeout, Transport: transport}

	serviceConfig, err := config.NewConfigBuilder().
		SetServiceDetails(rtDetails).
		SetDryRun(false).
		SetHttpClient(httpClient).
		Build()
	if err != nil {
		return nil, err
	}
	manager, err := artifactory.New(serviceConfig)
	if err != nil {
		return nil, err
	}

	logging.Info("Initializing Artifactory client...",
		"base_url", baseURL,
		"auth_method", string(creds.Method),
		"repo_name", creds.Repo,
		"ssl_verify", creds.SSLVerify,
		"timeout_seconds", creds.TimeoutSeconds,
		"max_retries", creds.MaxRetries,
	)

	return &Client{
		manager:    manager,
		baseURL:    baseURL,
		repo:       creds.Repo,
		httpClient: httpClient,
		limiter:    ratelimit.New(creds.RateLimit, time.Second),
		creds:      creds,
		retry:      newRetryLogic(creds.MaxRetries, creds.TimeoutSeconds),
	}, nil
}

func (c *Client) Repo() string    { return c.repo }
func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) repoPath(rel string) string {
	rel = strings.TrimLeft(rel, "/")
	if rel == "" {
		return c.repo
	}
	return c.repo + "/" + rel
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.limiter.Acquire()
	switch c.creds.Method {
	case auth.MethodAPIKey:
		req.Header.Set("X-JFrog-Art-Api", c.creds.APIKey)
	case auth.MethodToken:
		req.Header.Set("Authorization", "Bearer "+c.creds.Token)
	case auth.MethodBasic:
		req.SetBasicAuth(c.creds.Username, c.creds.Password)
	}
	// URL is validated at client construction (scheme/host); destination is the configured Artifactory instance.
	return c.httpClient.Do(req) //nolint:gosec // G704: intentional request to user-configured Artifactory URL
}

// FileExists uses the jfrog SDK FileInfo API.
func (c *Client) FileExists(ctx context.Context, relPath string) (bool, error) {
	var exists bool
	err := c.retry.executeWithRetry(ctx, func() error {
		c.limiter.Acquire()
		_, err := c.manager.FileInfo(c.repoPath(relPath))
		if err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "404") || strings.Contains(msg, "not found") {
				exists = false
				return nil
			}
			return err
		}
		exists = true
		return nil
	}, "FileExists")
	if err != nil {
		logging.Warn("Error checking file existence", "path", relPath, "error", err)
		return false, nil
	}
	return exists, nil
}

// GetFileInfo uses the jfrog SDK FileInfo API.
func (c *Client) GetFileInfo(ctx context.Context, relPath string) (*FileMeta, error) {
	var meta *FileMeta
	err := c.retry.executeWithRetry(ctx, func() error {
		c.limiter.Acquire()
		info, err := c.manager.FileInfo(c.repoPath(relPath))
		if err != nil {
			return err
		}
		size, _ := strconv.ParseInt(info.Size, 10, 64)
		meta = &FileMeta{
			Size:         size,
			SHA1:         info.Checksums.Sha1,
			MD5:          info.Checksums.Md5,
			LastModified: info.LastModified,
		}
		return nil
	}, "GetFileInfo")
	if err != nil {
		logging.Error("Error getting file info", "path", relPath, "error", err)
		return nil, err
	}
	return meta, nil
}

// ListItems uses the jfrog SDK FolderInfo API.
func (c *Client) ListItems(ctx context.Context, relPath string) ([]ChildItem, error) {
	var out []ChildItem
	err := c.retry.executeWithRetry(ctx, func() error {
		c.limiter.Acquire()
		info, err := c.manager.FolderInfo(c.repoPath(relPath))
		if err != nil {
			return err
		}
		out = make([]ChildItem, 0, len(info.Children))
		for _, ch := range info.Children {
			out = append(out, ChildItem{URI: ch.Uri, Folder: ch.Folder})
		}
		return nil
	}, "ListItems")
	if err != nil {
		logging.Error("Error listing items", "path", relPath, "error", err)
		return nil, err
	}
	return out, nil
}

// Download downloads file content via authenticated HTTP.
func (c *Client) Download(ctx context.Context, relPath string) ([]byte, error) {
	var data []byte
	err := c.retry.executeWithRetry(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+c.repoPath(relPath), nil)
		if err != nil {
			return err
		}
		resp, err := c.do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("download failed: %s", resp.Status)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		data = body
		return nil
	}, "Download")
	if err != nil {
		logging.Error("Error downloading file.", "path", relPath, "error", err)
		return nil, err
	}
	return data, nil
}

// Upload uploads content using jfrog UploadFiles with HTTP PUT fallback.
func (c *Client) Upload(ctx context.Context, relPath string, content []byte, contentType string) error {
	err := c.retry.executeWithRetry(ctx, func() error {
		tmp, err := os.CreateTemp("", "artifactory-content-library-upload-*")
		if err != nil {
			return err
		}
		tmpName := tmp.Name()
		defer func(name string) {
			_ = os.Remove(name)
		}(tmpName)
		if _, werr := tmp.Write(content); werr != nil {
			_ = tmp.Close()
			return werr
		}
		if err := tmp.Close(); err != nil {
			return err
		}

		params := services.NewUploadParams()
		params.Pattern = tmpName
		params.Target = strings.TrimRight(c.repoPath(relPath), "/")
		params.Flat = true

		c.limiter.Acquire()
		uploaded, failed, err := c.manager.UploadFiles(artifactory.UploadServiceOptions{}, params)
		if err == nil && failed == 0 && uploaded > 0 {
			return nil
		}

		req, err2 := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/"+c.repoPath(relPath), bytes.NewReader(content))
		if err2 != nil {
			if err != nil {
				return err
			}
			return err2
		}
		if contentType == "" {
			contentType = "application/json; charset=utf-8"
		}
		req.Header.Set("Content-Type", contentType)
		resp, err2 := c.do(req)
		if err2 != nil {
			return err2
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("upload failed: %s", resp.Status)
		}
		return nil
	}, "Upload")
	if err != nil {
		logging.Error("Error uploading file", "path", relPath, "error", err)
		return err
	}
	return nil
}

// Delete deletes a file via HTTP DELETE.
func (c *Client) Delete(ctx context.Context, relPath string) error {
	err := c.retry.executeWithRetry(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/"+c.repoPath(relPath), nil)
		if err != nil {
			return err
		}
		resp, err := c.do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("delete failed: %s", resp.Status)
		}
		return nil
	}, "Delete")
	if err != nil {
		logging.Error("Error deleting file", "path", relPath, "error", err)
		return err
	}
	logging.Audit("Deleted file", "path", relPath)
	return nil
}

// SetProperties sets item properties via the Artifactory metadata API.
func (c *Client) SetProperties(ctx context.Context, relPath string, props map[string]string) error {
	parts := make([]string, 0, len(props))
	for k, v := range props {
		parts = append(parts, k+"="+v)
	}
	url := c.baseURL + "/api/metadata/" + c.repoPath(relPath) + "?properties=" + strings.Join(parts, "|")
	return c.retry.executeWithRetry(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
		if err != nil {
			return err
		}
		resp, err := c.do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("set properties failed: %s", resp.Status)
		}
		return nil
	}, "SetProperties")
}

// MarshalIndent is a helper for JSON uploads.
func MarshalIndent(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
