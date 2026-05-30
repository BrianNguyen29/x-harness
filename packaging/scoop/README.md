# Scoop Manifest

This directory contains the generated Scoop manifest for installing x-harness on Windows via [Scoop](https://scoop.sh/).

## Manifest Generation

The manifest is generated automatically in CI by `scripts/generate-scoop-manifest.sh`.

### Local generation

To generate the manifest locally from a release checksums file:

```bash
./scripts/generate-scoop-manifest.sh v0.99.0-rc1 .x-harness/release/go-binaries/checksums.txt > packaging/scoop/x-harness.json
```

### Inputs

- `version` — the Git release tag (e.g. `v0.99.0-rc1`).
- `checksums.txt` — the SHA256 checksums file produced by the release workflow for all Go binaries.

The script extracts the Windows `amd64` and `arm64` hashes and emits a Scoop manifest with:

- `64bit` and `arm64` architecture blocks
- `checkver` pointing at the GitHub repository for auto-update discovery
- `autoupdate` templates using the GitHub Release asset URLs

## Bucket Update Process

This repository does **not** host a separate Scoop bucket. To publish the manifest:

1. Copy the generated `x-harness.json` into a Scoop bucket repository (e.g. `scoop bucket add x-harness <bucket-repo-url>`).
2. Commit and push the updated manifest to the bucket repo.
3. Users can then run `scoop update x-harness` to receive the new version.

### CI Artifact

The release workflow writes the manifest to `.x-harness/release/scoop/x-harness.json` and uploads it as part of the release artifacts. You can download it directly from the GitHub Release page and paste it into a bucket repo.

## Manual Fallback

If the manifest is not yet in a bucket, users can install directly from the release asset:

```powershell
# Download the latest Windows amd64 binary
Invoke-WebRequest -Uri "https://github.com/BrianNguyen29/x-harness/releases/latest/download/x-harness-<version>-windows-amd64.exe" -OutFile "x-harness.exe"
```

Replace `<version>` with the desired release tag.
