---
icon: octicons/gear-16
---

# Configuration

Configuration files are strictly decoded and validated for the command in use.

Prefer `${ENV}` references for secrets; plaintext secret values produce a warning.

## Priority

**CLI Flags > YAML > Environment Variables > Defaults**

## Example Configurations

Artifactory (`generate`):

```yaml
--8<-- "example/configuration/example-artifactory-config.yaml"
```

Subscribe (`subscribe`):

```yaml
--8<-- "example/configuration/example-subscribe-config.yaml"
```

## Schema

A JSON Schema for configuration files is published with the project at
[`schema/config.schema.json`](https://github.com/tenthirtyam/artifactory-content-library/blob/main/schema/config.schema.json).

## Reference

See [Reference](reference.md) for global options, command flags, and environment
variables.
