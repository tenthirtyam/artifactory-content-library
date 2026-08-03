---
icon: octicons/verified-16
---

# Release Verification

Release archives are published on
[GitHub Releases](https://github.com/tenthirtyam/artifactory-content-library/releases).

Each release includes platform archives, a `SHA256SUMS` checksum file, and a
detached PGP signature (`.sig`).

Only the `SHA256SUMS` file is signed. The archives themselves are not signed, but
are hashed. To verify the integrity of a particular archive:

1. Download the archive, `SHA256SUMS`, and `SHA256SUMS.sig` files from the
   release.
2. Verify the `SHA256SUMS` file is properly signed.
3. Verify the checksum in the file matches the archive.

## PGP Public Key

| Attribute       | Details                                            |
| --------------- | -------------------------------------------------- |
| **Key Name**    | release-bot (GitHub Release Signing)               |
| **Email**       | release@tenthirtyam.org                            |
| **Fingerprint** | D8FE BF37 A81D CFA5 226B 6F24 2B72 08F7 BFE2 440C |
| **Key ID**      | BFE2440C                                           |
| **Long Key ID** | 2B7208F7BFE2440C                                   |

The public key can be obtained from
[keys.openpgp.org](https://keys.openpgp.org/vks/v1/by-fingerprint/D8FEBF37A81DCFA5226B6F242B7208F7BFE2440C).

## Example

The following example verifies a release archive.

Substitute `VERSION`, `OS`, and `ARCH` for the target release. Windows archives use
`.zip`; other platforms use `.tar.gz`.

```sh
# Set the release target and derived paths.
FINGERPRINT=D8FEBF37A81DCFA5226B6F242B7208F7BFE2440C
VERSION=1.0.0
OS=linux
ARCH=amd64
EXT=tar.gz
BASE_URL="https://github.com/tenthirtyam/artifactory-content-library/releases/download/v${VERSION}"
PREFIX="artifactory-content-library_v${VERSION}"
ARCHIVE="artifactory-content-library_${VERSION}_${OS}_${ARCH}.${EXT}"
CHECKSUMS="${PREFIX}_SHA256SUMS"
CHECKSUMS_SIG="${CHECKSUMS}.sig"

# Import the public key.
gpg --keyserver keys.openpgp.org --recv-keys "${FINGERPRINT}"

# Download the archive and signature files.
curl -LO "${BASE_URL}/${ARCHIVE}"
curl -LO "${BASE_URL}/${CHECKSUMS}"
curl -LO "${BASE_URL}/${CHECKSUMS_SIG}"

# Verify the signature file is untampered.
gpg --verify "${CHECKSUMS_SIG}" "${CHECKSUMS}"

# Verify the checksum matches the archive.
shasum -a 256 -c "${CHECKSUMS}" --ignore-missing
```

## Expected Output

A successful signature verification reports a good signature:

```text
gpg: Good signature from "release-bot (GitHub Release Signing) <release@tenthirtyam.org>" [unknown]
gpg: WARNING: This key is not certified with a trusted signature!
gpg:          There is no indication that the signature belongs to the owner.
```

!!! note
    The trust warning is **normal** for a freshly imported key. It means GPG
    cannot confirm the key owner's identity through its web of trust, not that
    the signature failed. As long as you imported the key using the fingerprint
    above and the output reports `Good signature`, the checksum file is
    authentic.

A successful checksum verification ends with the following structure:

```text
artifactory-content-library_<version>_<os>_<arch>.tar.gz: OK
```

## Mark the Key as Trusted (Optional)

To trust the key for future verifications, confirm the fingerprint and set a
local trust level:

```sh
FINGERPRINT=D8FEBF37A81DCFA5226B6F242B7208F7BFE2440C

gpg --fingerprint "${FINGERPRINT}"
gpg --edit-key "${FINGERPRINT}"
```

At the `gpg>` prompt, run `trust`, choose `4` (I trust fully), confirm with `y`,
then run `quit`.
