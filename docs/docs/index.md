---
icon: octicons/home-16
---

# Artifactory Content Library for VMware vSphere

Generate metadata for content stored in a JFrog Artifactory repository for use as a VMware vSphere
content library.

## Features

- **Artifactory**: Generate metadata for content stored in JFrog Artifactory.
- **Content Type Detection**: Automatic detection of OVF, OVA, and ISO content.
- **Incremental Updates**: Change detection with checksum validation.
- **Dry Run**: Preview added, removed, and changed items without writing metadata.
- **Subscribed Library Helper**: Create subscribed content libraries in vSphere with an Artifactory
  repository as the published content library source.

## Requirements

- Go 1.26.5 (to build from source)
- VMware vSphere 8.0.3 or higher (for subscribed libraries)

## Next Steps

1. [Install](getting-started/installation.md) the CLI.
2. Arrange content using the [content layout](getting-started/content-layout.md) rules.
3. [Generate](getting-started/generate.md) library metadata in Artifactory.
4. Optionally [subscribe](getting-started/subscribe.md) a vSphere content library.
5. Review [configuration](configuration/index.md) and the
   [CLI reference](configuration/reference.md).
