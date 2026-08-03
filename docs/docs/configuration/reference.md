---
icon: octicons/book-16
---

# Reference

## Global Options

```bash
artifactory-content-library [command] [options]
```

| Option         | Type     | Description                                        | Default    |
| -------------- | -------- | -------------------------------------------------- | ---------- |
| `--help`       | -        | Show Help                                          | -          |
| `--version`    | -        | Show Version                                       | -          |
| `--log-format` | `string` | Logging Format: `standard` or `structured`         | `standard` |
| `--log-level`  | `string` | Logging Level: `debug`, `info`, `warning`, `error` | `info`     |

## `init` Command Options

```bash
artifactory-content-library init [options]
```

| Option     | Type     | Description                                | Default       |
| ---------- | -------- | ------------------------------------------ | ------------- |
| `--output` | `string` | Path for the example configuration file    | -             |
| `--type`   | `string` | Example type: `artifactory` or `subscribe` | `artifactory` |
| `--force`  | `bool`   | Overwrite an existing configuration file   | `false`       |

## `generate` Command Options

```bash
artifactory-content-library generate [options]
```

| Option              | Type     | Description                              | Default | Environment Variables         |
| ------------------- | -------- | ---------------------------------------- | ------- | ----------------------------- |
| `--config`          | `string` | Configuration File                       | -       | -                             |
| `--name`            | `string` | Content Library Name                     | -       | -                             |
| `--path`            | `string` | Artifactory Content Base Path            | -       | -                             |
| `--skip-cert`       | `bool`   | Skip Certificate files in OVF Packages   | `true`  | -                             |
| `--url`             | `string` | Artifactory Base URL                     | -       | `ARTIFACTORY_URL`             |
| `--repo`            | `string` | Artifactory Repository                   | -       | `ARTIFACTORY_REPOSITORY`      |
| `--username`        | `string` | Artifactory Username (with `--password`) | -       | `ARTIFACTORY_USERNAME`        |
| `--password`        | `string` | Artifactory Password (with `--username`) | -       | `ARTIFACTORY_PASSWORD`        |
| `--api-key`         | `string` | Artifactory API Key                      | -       | `ARTIFACTORY_API_KEY`         |
| `--token`           | `string` | Artifactory Access Token                 | -       | `ARTIFACTORY_TOKEN`           |
| `--ssl-verify`      | `bool`   | Verify SSL Certificate (`ssl_verify`)    | `true`  | `ARTIFACTORY_SSL_VERIFY`      |
| `--rate-limit`      | `int`    | Requests per Second (`rate_limit`)       | `10`    | `ARTIFACTORY_RATE_LIMIT`      |
| `--timeout-seconds` | `int`    | HTTP Client Timeout in Seconds           | `30`    | `ARTIFACTORY_TIMEOUT_SECONDS` |
| `--max-retries`     | `int`    | Maximum Retries                          | `3`     | `ARTIFACTORY_MAX_RETRIES`     |

## `subscribe` Command Options

```bash
artifactory-content-library subscribe [options]
```

| Option                         | Type     | Description                                         | Default | Environment Variables                |
| ------------------------------ | -------- | --------------------------------------------------- | ------- | ------------------------------------ |
| `--config`                     | `string` | Configuration File                                  | -       | -                                    |
| `--url`                        | `string` | vSphere URL                                         | -       | `VSPHERE_URL`                        |
| `--username`                   | `string` | vSphere Username                                    | -       | `VSPHERE_USERNAME`                   |
| `--password`                   | `string` | vSphere Password                                    | -       | `VSPHERE_PASSWORD`                   |
| `--ssl-verify`                 | `bool`   | Verify SSL Certificate                              | `true`  | `VSPHERE_SSL_VERIFY`                 |
| `--name`                       | `string` | Subscribed Library Name                             | -       | `VSPHERE_LIBRARY_NAME`               |
| `--datacenter`                 | `string` | Datacenter Name                                     | -       | `VSPHERE_DATACENTER`                 |
| `--datastore`                  | `string` | Datastore Name                                      | -       | `VSPHERE_LIBRARY_DATASTORE`          |
| `--auto-sync`                  | `bool`   | Enable Automatic Sync                               | `false` | `VSPHERE_LIBRARY_AUTO_SYNC`          |
| `--on-demand`                  | `bool`   | On-demand Content Download                          | `false` | `VSPHERE_LIBRARY_ON_DEMAND`          |
| `--publisher-subscription-url` | `string` | Publisher Subscription URL                          | -       | `VSPHERE_PUBLISHER_SUBSCRIPTION_URL` |
| `--publisher-ssl-thumbprint`   | `string` | Publisher SSL Thumbprint                            | -       | `VSPHERE_PUBLISHER_SSL_THUMBPRINT`   |
| `--publisher-username`         | `string` | Artifactory Username (BASIC auth for publisher URL) | -       | `VSPHERE_PUBLISHER_USERNAME`         |
| `--publisher-password`         | `string` | Artifactory Password (BASIC auth for publisher URL) | -       | `VSPHERE_PUBLISHER_PASSWORD`         |

## Artifactory Environment Variables

| Variable                      | Use                            |
| ----------------------------- | ------------------------------ |
| `ARTIFACTORY_URL`             | Artifactory Base URL           |
| `ARTIFACTORY_REPOSITORY`      | Artifactory Repository         |
| `ARTIFACTORY_USERNAME`        | Artifactory Username           |
| `ARTIFACTORY_PASSWORD`        | Artifactory Password           |
| `ARTIFACTORY_API_KEY`         | Artifactory API Key            |
| `ARTIFACTORY_TOKEN`           | Artifactory Access Token       |
| `ARTIFACTORY_SSL_VERIFY`      | Verify SSL Certificate         |
| `ARTIFACTORY_RATE_LIMIT`      | Requests per Second            |
| `ARTIFACTORY_TIMEOUT_SECONDS` | HTTP Client Timeout in Seconds |
| `ARTIFACTORY_MAX_RETRIES`     | Maximum Retries                |

Use exactly one authentication method.

## vSphere Environment Variables

| Variable                             | Use                                         |
| ------------------------------------ | ------------------------------------------- |
| `VSPHERE_URL`                        | vSphere URL                                 |
| `VSPHERE_USERNAME`                   | vSphere Username                            |
| `VSPHERE_PASSWORD`                   | vSphere Password                            |
| `VSPHERE_SSL_VERIFY`                 | Verify SSL Certificate                      |
| `VSPHERE_DATACENTER`                 | Datacenter Name                             |
| `VSPHERE_LIBRARY_NAME`               | Subscribed Library Name                     |
| `VSPHERE_LIBRARY_DATASTORE`          | Datastore Name                              |
| `VSPHERE_LIBRARY_AUTO_SYNC`          | Enable Automatic Sync                       |
| `VSPHERE_LIBRARY_ON_DEMAND`          | On-demand Content Download                  |
| `VSPHERE_PUBLISHER_SUBSCRIPTION_URL` | Publisher Subscription URL                  |
| `VSPHERE_PUBLISHER_SSL_THUMBPRINT`   | Publisher SSL Thumbprint                    |
| `VSPHERE_PUBLISHER_USERNAME`         | Artifactory Username (Basic Authentication) |
| `VSPHERE_PUBLISHER_PASSWORD`         | Artifactory Password (Basic Authentication) |

Priority: **CLI Flags > YAML > Environment Variables > Defaults**.
