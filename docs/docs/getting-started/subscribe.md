---
icon: octicons/sync-16
---

# Subscribe

The `subscribe` command creates a subscribed content library in vSphere that pulls from
an Artifactory-published `lib.json` URL.

vSphere needs Artifactory basic authentication credentials to pull content from a
protected repository.

## CLI Example

Refer to [`subscribe` command options](../configuration/reference.md#subscribe-command-options)
for the full flag list.

```bash
artifactory-content-library subscribe \
  --url "https://vc01.example.com" \
  --username "administrator@vsphere.local" \
  --password "password" \
  --ssl-verify true \
  --datacenter "dc01" \
  --name "artifactory-library" \
  --datastore "nfs" \
  --publisher-subscription-url "https://packages.example.com/artifactory/example-repository/example-library/lib.json" \
  --publisher-username "admin" \
  --publisher-password "password"
```

## Environment Variables

Refer to [vSphere environment variables](../configuration/reference.md#vsphere-environment-variables)
for the full list.

```bash
export VSPHERE_URL="https://vc01.example.com"
export VSPHERE_USERNAME="administrator@vsphere.local"
export VSPHERE_PASSWORD="password"
export VSPHERE_SSL_VERIFY="true"
export VSPHERE_PUBLISHER_USERNAME="admin"
export VSPHERE_PUBLISHER_PASSWORD="password"

artifactory-content-library subscribe \
  --name "artifactory-library" \
  --datacenter "dc01" \
  --datastore "nfs" \
  --publisher-subscription-url "https://packages.example.com/artifactory/example-repository/example-library/lib.json"
```

## Configuration File

Refer to [`init` command options](../configuration/reference.md#init-command-options)
for the full flag list.

Generate an example subscribe configuration:

```bash
artifactory-content-library init --type subscribe --output subscribe.yaml
```

Example:

```yaml
--8<-- "docs/snippets/example-subscribe-config.yaml"
```

```bash
artifactory-content-library subscribe --config subscribe.yaml
```
