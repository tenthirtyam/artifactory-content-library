// SPDX-License-Identifier: MIT
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tenthirtyam/artifactory-content-library/internal/artifactory"
	"github.com/tenthirtyam/artifactory-content-library/internal/auth"
	"github.com/tenthirtyam/artifactory-content-library/internal/config"
	"github.com/tenthirtyam/artifactory-content-library/internal/logging"
	"github.com/tenthirtyam/artifactory-content-library/internal/vsphere"
)

// Build metadata injected via ldflags / SetVersion.
var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

// SetVersion records release metadata for --version output.
func SetVersion(version, commit, date string) {
	if version != "" {
		buildVersion = version
	}
	if commit != "" {
		buildCommit = commit
	}
	if date != "" {
		buildDate = date
	}
}

// NewRootCommand builds the artifactory-content-library CLI.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "artifactory-content-library",
		Short:        "Generate vSphere content library metadata for JFrog Artifactory",
		Long:         "Artifactory Content Library — generate vSphere content library metadata in JFrog Artifactory.",
		Version:      fmt.Sprintf("%s (commit=%s date=%s)", buildVersion, buildCommit, buildDate),
		SilenceUsage: true,
	}

	root.PersistentFlags().String("log-format", "standard", "Logging format: standard or structured")
	root.PersistentFlags().String("log-level", "info", "Logging level: debug, info, warning, error")
	_ = viper.BindPFlag("logging.format", root.PersistentFlags().Lookup("log-format"))
	_ = viper.BindPFlag("logging.level", root.PersistentFlags().Lookup("log-level"))

	bindEnv()

	root.AddCommand(newInitCmd())
	root.AddCommand(newGenerateCmd())
	root.AddCommand(newSubscribeCmd())

	return root
}

func bindEnv() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = viper.BindEnv("artifactory.url", "ARTIFACTORY_URL")
	_ = viper.BindEnv("artifactory.repo", "ARTIFACTORY_REPOSITORY")
	_ = viper.BindEnv("artifactory.auth.api_key", "ARTIFACTORY_API_KEY")
	_ = viper.BindEnv("artifactory.auth.username", "ARTIFACTORY_USERNAME")
	_ = viper.BindEnv("artifactory.auth.password", "ARTIFACTORY_PASSWORD")
	_ = viper.BindEnv("artifactory.auth.token", "ARTIFACTORY_TOKEN")
	_ = viper.BindEnv("artifactory.ssl_verify", "ARTIFACTORY_SSL_VERIFY")
	_ = viper.BindEnv("artifactory.rate_limit", "ARTIFACTORY_RATE_LIMIT")
	_ = viper.BindEnv("artifactory.timeout_seconds", "ARTIFACTORY_TIMEOUT_SECONDS")
	_ = viper.BindEnv("artifactory.max_retries", "ARTIFACTORY_MAX_RETRIES")

	_ = viper.BindEnv("vsphere.url", "VSPHERE_URL")
	_ = viper.BindEnv("vsphere.username", "VSPHERE_USERNAME")
	_ = viper.BindEnv("vsphere.password", "VSPHERE_PASSWORD")
	_ = viper.BindEnv("vsphere.ssl_verify", "VSPHERE_SSL_VERIFY")

	_ = viper.BindEnv("library.name", "VSPHERE_LIBRARY_NAME")
	_ = viper.BindEnv("library.datacenter", "VSPHERE_DATACENTER")
	_ = viper.BindEnv("library.datastore", "VSPHERE_LIBRARY_DATASTORE")
	_ = viper.BindEnv("library.auto_sync", "VSPHERE_LIBRARY_AUTO_SYNC")
	_ = viper.BindEnv("library.on_demand", "VSPHERE_LIBRARY_ON_DEMAND")
	_ = viper.BindEnv("library.publisher_subscription_url", "VSPHERE_PUBLISHER_SUBSCRIPTION_URL")
	_ = viper.BindEnv("library.publisher_ssl_thumbprint", "VSPHERE_PUBLISHER_SSL_THUMBPRINT")
	_ = viper.BindEnv("library.publisher_username", "VSPHERE_PUBLISHER_USERNAME")
	_ = viper.BindEnv("library.publisher_password", "VSPHERE_PUBLISHER_PASSWORD")

}

func addGenerateFlags(cmd *cobra.Command) {
	cmd.Flags().String("config", "", "Path to YAML configuration file")
	cmd.Flags().String("name", "", "Content library name")
	cmd.Flags().String("path", "", "Artifactory content base path")
	cmd.Flags().String("skip-cert", "true", "Skip certificate files in OVF packages (true/false)")
	cmd.Flags().String("url", "", "Artifactory base URL")
	cmd.Flags().String("repo", "", "Artifactory repository name")
	cmd.Flags().String("api-key", "", "Artifactory API key (exclusive auth method)")
	cmd.Flags().String("username", "", "Artifactory username (with --password)")
	cmd.Flags().String("password", "", "Artifactory password (with --username)")
	cmd.Flags().String("token", "", "Artifactory access token (exclusive auth method)")
	cmd.Flags().String("ssl-verify", "", "TLS verify true/false (default true)")
	cmd.Flags().String("rate-limit", "", "Artifactory requests per second (default 10)")
	cmd.Flags().String("timeout-seconds", "", "Artifactory HTTP timeout in seconds (default 30)")
	cmd.Flags().String("max-retries", "", "Artifactory request retries (default 3)")

	_ = viper.BindPFlag("config", cmd.Flags().Lookup("config"))
	_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))
	_ = viper.BindPFlag("path", cmd.Flags().Lookup("path"))
	_ = viper.BindPFlag("skip_cert", cmd.Flags().Lookup("skip-cert"))
}

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate content library metadata (one-shot)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			applyChangedArtifactoryFlags(cmd)
			configureLogging()
			return runGenerate(cmd.Context())
		},
	}
	addGenerateFlags(cmd)
	return cmd
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write an example configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, _ := cmd.Flags().GetString("output")
			t, _ := cmd.Flags().GetString("type")
			force, _ := cmd.Flags().GetBool("force")
			if out == "" {
				return fmt.Errorf("--output is required")
			}
			return runConfigInit(out, t, force)
		},
	}
	cmd.Flags().String("output", "", "Output file path")
	cmd.Flags().String("type", "artifactory", "Example type: artifactory or subscribe")
	cmd.Flags().Bool("force", false, "Overwrite an existing configuration file")
	return cmd
}

func newSubscribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe",
		Short: "Create a subscribed content library in vSphere",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			applyChangedSubscribeFlags(cmd)
			configureLogging()
			return runSubscribe(cmd.Context())
		},
	}
	cmd.Flags().String("config", "", "YAML configuration file")
	cmd.Flags().String("url", "", "vSphere URL")
	cmd.Flags().String("username", "", "vSphere username")
	cmd.Flags().String("password", "", "vSphere password")
	cmd.Flags().String("ssl-verify", "", "TLS verify true/false (default true)")
	cmd.Flags().String("name", "", "Library name")
	cmd.Flags().String("datacenter", "", "Datacenter name (required when more than one exists)")
	cmd.Flags().String("datastore", "", "Datastore name")
	cmd.Flags().Bool("auto-sync", false, "Enable automatic sync")
	cmd.Flags().Bool("on-demand", false, "On-demand content download")
	cmd.Flags().String("publisher-subscription-url", "", "Publisher subscription URL")
	cmd.Flags().String("publisher-ssl-thumbprint", "", "Publisher SSL thumbprint")
	cmd.Flags().String("publisher-username", "", "Publisher username")
	cmd.Flags().String("publisher-password", "", "Publisher password")

	_ = viper.BindPFlag("config", cmd.Flags().Lookup("config"))
	return cmd
}

func configureLogging() {
	logging.Configure(viper.GetString("logging.format"), viper.GetString("logging.level"))
}

func runConfigInit(path, storageType string, force bool) error {
	if storageType == "" {
		storageType = "artifactory"
	}
	switch storageType {
	case "artifactory", "subscribe":
	default:
		return fmt.Errorf("--type must be artifactory or subscribe")
	}
	return config.WriteExample(path, storageType, force)
}

func runGenerate(ctx context.Context) error {
	cfgPath := viper.GetString("config")
	if cfgPath != "" {
		return runFromConfig(ctx, cfgPath)
	}

	name := viper.GetString("name")
	path := viper.GetString("path")
	skipCert := auth.ParseBoolEnv(fmt.Sprint(viper.Get("skip_cert")), true)

	if name == "" {
		return fmt.Errorf("--name is required")
	}

	return generateOne(ctx, name, path, skipCert, nil)
}

func runFromConfig(ctx context.Context, cfgPath string) error {
	cfg, warnings, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		logging.Warn(w)
	}
	if err := cfg.ValidateForGenerate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	if cfg.Logging.Format != "" {
		viper.Set("logging.format", cfg.Logging.Format)
	}
	if cfg.Logging.Level != "" {
		viper.Set("logging.level", cfg.Logging.Level)
	}
	configureLogging()

	target := viper.GetString("name")
	libs := cfg.Libraries
	if target != "" {
		filtered := libs[:0]
		for _, lib := range libs {
			if lib.Name == target {
				filtered = append(filtered, lib)
			}
		}
		libs = filtered
		if len(libs) == 0 {
			return fmt.Errorf("content library %q not found in configuration file", target)
		}
	}

	var firstErr error
	for _, lib := range libs {
		skipCert := config.BoolVal(lib.SkipCert, config.BoolVal(cfg.Defaults.SkipCert, true))

		if err := generateOne(ctx, lib.Name, lib.Path, skipCert, lib.Artifactory); err != nil {
			logging.Error("Failed to generate content library", "library_name", lib.Name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		logging.Info("Successfully generated content library", "library_name", lib.Name)
	}
	return firstErr
}

func generateOne(ctx context.Context, name, path string, skipCert bool, artCfg *config.Artifactory) error {
	creds, err := resolveArtifactoryCreds(artCfg)
	if err != nil {
		return err
	}
	client, err := artifactory.NewClient(creds)
	if err != nil {
		return err
	}
	return artifactory.Generate(ctx, client, name, path, skipCert)
}

// applyChangedArtifactoryFlags records only explicitly set generate flags so they
// outrank YAML and environment values (flags > config > env).
func applyChangedArtifactoryFlags(cmd *cobra.Command) {
	set := func(flag, key string) {
		if !cmd.Flags().Changed(flag) {
			return
		}
		v, _ := cmd.Flags().GetString(flag)
		viper.Set(key, v)
	}
	set("url", "flag.artifactory.url")
	set("repo", "flag.artifactory.repo")
	set("api-key", "flag.artifactory.auth.api_key")
	set("username", "flag.artifactory.auth.username")
	set("password", "flag.artifactory.auth.password")
	set("token", "flag.artifactory.auth.token")
	set("ssl-verify", "flag.artifactory.ssl_verify")
	set("rate-limit", "flag.artifactory.rate_limit")
	set("timeout-seconds", "flag.artifactory.timeout_seconds")
	set("max-retries", "flag.artifactory.max_retries")
}

func resolveArtifactoryCreds(artCfg *config.Artifactory) (*auth.Credentials, error) {
	// Priority: CLI flags > YAML > environment variables.
	cfg := auth.Config{
		URL:            os.Getenv("ARTIFACTORY_URL"),
		Repo:           os.Getenv("ARTIFACTORY_REPOSITORY"),
		APIKey:         os.Getenv("ARTIFACTORY_API_KEY"),
		Username:       os.Getenv("ARTIFACTORY_USERNAME"),
		Password:       os.Getenv("ARTIFACTORY_PASSWORD"),
		Token:          os.Getenv("ARTIFACTORY_TOKEN"),
		RateLimit:      auth.ParseIntEnv(os.Getenv("ARTIFACTORY_RATE_LIMIT"), auth.DefaultRateLimit),
		TimeoutSeconds: auth.ParseIntEnv(os.Getenv("ARTIFACTORY_TIMEOUT_SECONDS"), auth.DefaultTimeoutSeconds),
		MaxRetries:     auth.ParseIntEnv(os.Getenv("ARTIFACTORY_MAX_RETRIES"), auth.DefaultMaxRetries),
	}
	if v, ok := os.LookupEnv("ARTIFACTORY_SSL_VERIFY"); ok {
		b := auth.ParseBoolEnv(v, true)
		cfg.SSLVerify = &b
	}

	if artCfg != nil {
		if artCfg.URL != "" {
			cfg.URL = artCfg.URL
		}
		if artCfg.Repo != "" {
			cfg.Repo = artCfg.Repo
		}
		if artCfg.Auth.APIKey != "" {
			cfg.APIKey = artCfg.Auth.APIKey
		}
		if artCfg.Auth.Username != "" {
			cfg.Username = artCfg.Auth.Username
		}
		if artCfg.Auth.Password != "" {
			cfg.Password = artCfg.Auth.Password
		}
		if artCfg.Auth.Token != "" {
			cfg.Token = artCfg.Auth.Token
		}
		if artCfg.SSLVerify != nil {
			cfg.SSLVerify = artCfg.SSLVerify
		}
		if artCfg.RateLimit > 0 {
			cfg.RateLimit = artCfg.RateLimit
		}
		if artCfg.TimeoutSeconds > 0 {
			cfg.TimeoutSeconds = artCfg.TimeoutSeconds
		}
		if artCfg.MaxRetries > 0 {
			cfg.MaxRetries = artCfg.MaxRetries
		}
	}

	if v := viper.GetString("flag.artifactory.url"); v != "" {
		cfg.URL = v
	}
	if v := viper.GetString("flag.artifactory.repo"); v != "" {
		cfg.Repo = v
	}
	if v := viper.GetString("flag.artifactory.auth.api_key"); v != "" {
		cfg.APIKey = v
	}
	if v := viper.GetString("flag.artifactory.auth.username"); v != "" {
		cfg.Username = v
	}
	if v := viper.GetString("flag.artifactory.auth.password"); v != "" {
		cfg.Password = v
	}
	if v := viper.GetString("flag.artifactory.auth.token"); v != "" {
		cfg.Token = v
	}
	if viper.IsSet("flag.artifactory.ssl_verify") {
		b := auth.ParseBoolEnv(viper.GetString("flag.artifactory.ssl_verify"), true)
		cfg.SSLVerify = &b
	}
	if viper.IsSet("flag.artifactory.rate_limit") {
		cfg.RateLimit = auth.ParseIntEnv(viper.GetString("flag.artifactory.rate_limit"), cfg.RateLimit)
	}
	if viper.IsSet("flag.artifactory.timeout_seconds") {
		cfg.TimeoutSeconds = auth.ParseIntEnv(viper.GetString("flag.artifactory.timeout_seconds"), cfg.TimeoutSeconds)
	}
	if viper.IsSet("flag.artifactory.max_retries") {
		cfg.MaxRetries = auth.ParseIntEnv(viper.GetString("flag.artifactory.max_retries"), cfg.MaxRetries)
	}
	return auth.Resolve(cfg)
}

// applyChangedSubscribeFlags records only explicitly set subscribe flags so they
// outrank YAML and environment values (flags > config > env).
func applyChangedSubscribeFlags(cmd *cobra.Command) {
	setStr := func(flag, key string) {
		if !cmd.Flags().Changed(flag) {
			return
		}
		v, _ := cmd.Flags().GetString(flag)
		viper.Set(key, v)
	}
	setBool := func(flag, key string) {
		if !cmd.Flags().Changed(flag) {
			return
		}
		v, _ := cmd.Flags().GetBool(flag)
		viper.Set(key, v)
	}
	setStr("url", "flag.vsphere.url")
	setStr("username", "flag.vsphere.username")
	setStr("password", "flag.vsphere.password")
	setStr("ssl-verify", "flag.vsphere.ssl_verify")
	setStr("name", "flag.library.name")
	setStr("datacenter", "flag.library.datacenter")
	setStr("datastore", "flag.library.datastore")
	setBool("auto-sync", "flag.library.auto_sync")
	setBool("on-demand", "flag.library.on_demand")
	setStr("publisher-subscription-url", "flag.library.publisher_subscription_url")
	setStr("publisher-ssl-thumbprint", "flag.library.publisher_ssl_thumbprint")
	setStr("publisher-username", "flag.library.publisher_username")
	setStr("publisher-password", "flag.library.publisher_password")
}

func runSubscribe(ctx context.Context) error {
	cfgPath := viper.GetString("config")

	// Priority: CLI flags > YAML > environment variables > defaults.
	vsphereURL := os.Getenv("VSPHERE_URL")
	vsphereUsername := os.Getenv("VSPHERE_USERNAME")
	vspherePassword := os.Getenv("VSPHERE_PASSWORD")
	vsphereSSLVerify := true
	if v, ok := os.LookupEnv("VSPHERE_SSL_VERIFY"); ok {
		vsphereSSLVerify = auth.ParseBoolEnv(v, true)
	}
	libraryName := os.Getenv("VSPHERE_LIBRARY_NAME")
	libraryDatacenter := os.Getenv("VSPHERE_DATACENTER")
	libraryDatastore := os.Getenv("VSPHERE_LIBRARY_DATASTORE")
	libraryAutoSync := auth.ParseBoolEnv(os.Getenv("VSPHERE_LIBRARY_AUTO_SYNC"), false)
	libraryOnDemand := auth.ParseBoolEnv(os.Getenv("VSPHERE_LIBRARY_ON_DEMAND"), false)
	pubSubscriptionURL := os.Getenv("VSPHERE_PUBLISHER_SUBSCRIPTION_URL")
	pubSSLThumbprint := os.Getenv("VSPHERE_PUBLISHER_SSL_THUMBPRINT")
	pubUsername := os.Getenv("VSPHERE_PUBLISHER_USERNAME")
	pubPassword := os.Getenv("VSPHERE_PUBLISHER_PASSWORD")

	if cfgPath != "" {
		file, warnings, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		for _, w := range warnings {
			logging.Warn(w)
		}
		if err := file.ValidateForSubscribe(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
		if file.VSphere.URL != "" {
			vsphereURL = file.VSphere.URL
		}
		if file.VSphere.Username != "" {
			vsphereUsername = file.VSphere.Username
		}
		if file.VSphere.Password != "" {
			vspherePassword = file.VSphere.Password
		}
		if file.VSphere.SSLVerify != nil {
			vsphereSSLVerify = *file.VSphere.SSLVerify
		}
		if file.Library.Name != "" {
			libraryName = file.Library.Name
		}
		if file.Library.Datacenter != "" {
			libraryDatacenter = file.Library.Datacenter
		}
		if file.Library.Datastore != "" {
			libraryDatastore = file.Library.Datastore
		}
		libraryAutoSync = file.Library.AutoSync
		libraryOnDemand = file.Library.OnDemand
		if file.Library.PublisherSubscriptionURL != "" {
			pubSubscriptionURL = file.Library.PublisherSubscriptionURL
		}
		if file.Library.PublisherSSLThumbprint != "" {
			pubSSLThumbprint = file.Library.PublisherSSLThumbprint
		}
		if file.Library.PublisherUsername != "" {
			pubUsername = file.Library.PublisherUsername
		}
		if file.Library.PublisherPassword != "" {
			pubPassword = file.Library.PublisherPassword
		}
	}

	if v := viper.GetString("flag.vsphere.url"); v != "" {
		vsphereURL = v
	}
	if v := viper.GetString("flag.vsphere.username"); v != "" {
		vsphereUsername = v
	}
	if v := viper.GetString("flag.vsphere.password"); v != "" {
		vspherePassword = v
	}
	if viper.IsSet("flag.vsphere.ssl_verify") {
		vsphereSSLVerify = auth.ParseBoolEnv(viper.GetString("flag.vsphere.ssl_verify"), true)
	}
	if v := viper.GetString("flag.library.name"); v != "" {
		libraryName = v
	}
	if v := viper.GetString("flag.library.datacenter"); v != "" {
		libraryDatacenter = v
	}
	if v := viper.GetString("flag.library.datastore"); v != "" {
		libraryDatastore = v
	}
	if viper.IsSet("flag.library.auto_sync") {
		libraryAutoSync = viper.GetBool("flag.library.auto_sync")
	}
	if viper.IsSet("flag.library.on_demand") {
		libraryOnDemand = viper.GetBool("flag.library.on_demand")
	}
	if v := viper.GetString("flag.library.publisher_subscription_url"); v != "" {
		pubSubscriptionURL = v
	}
	if v := viper.GetString("flag.library.publisher_ssl_thumbprint"); v != "" {
		pubSSLThumbprint = v
	}
	if v := viper.GetString("flag.library.publisher_username"); v != "" {
		pubUsername = v
	}
	if v := viper.GetString("flag.library.publisher_password"); v != "" {
		pubPassword = v
	}

	_, err := vsphere.CreateSubscribedLibrary(ctx, vsphere.SubscribeConfig{
		URL:                      vsphereURL,
		Username:                 vsphereUsername,
		Password:                 vspherePassword,
		Insecure:                 !vsphereSSLVerify,
		Name:                     libraryName,
		Datacenter:               libraryDatacenter,
		Datastore:                libraryDatastore,
		AutoSync:                 libraryAutoSync,
		OnDemand:                 libraryOnDemand,
		PublisherSubscriptionURL: pubSubscriptionURL,
		PublisherSSLThumbprint:   pubSSLThumbprint,
		PublisherUsername:        pubUsername,
		PublisherPassword:        pubPassword,
	})
	return err
}

// Execute runs the root command.
func Execute() {
	ctx := context.Background()
	if err := NewRootCommand().ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
