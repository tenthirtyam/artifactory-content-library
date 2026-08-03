---
icon: octicons/download-16
---

# Installation

## Build from Source

```bash
git clone https://github.com/tenthirtyam/artifactory-content-library.git
cd artifactory-content-library

make build

# Or...

go build -o artifactory-content-library .
```

## Install from Source

```bash
go install github.com/tenthirtyam/artifactory-content-library@v0.1.0
```

## Verify

```bash
artifactory-content-library --version
```

Release archives are published on
[GitHub Releases](https://github.com/tenthirtyam/artifactory-content-library/releases).

Refer to [Release Verification](../community/release-verification.md) to verify checksums
and signatures.
