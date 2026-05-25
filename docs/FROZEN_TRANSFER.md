# Frozen Transfer

Frozen transfer packages the file-first harness contract for another workspace. The bundle is deterministic enough for review: it includes a manifest, per-file SHA-256 checksums, version metadata, schemas, policies, templates, adapters, docs, examples, components, and the experimental evolution constitution/budget.

## Export

```bash
node packages/cli/dist/index.js export --frozen --out dist/x-harness-frozen.tar.gz --json
```

Export writes a gzipped tar archive with:

- `manifest.json`
- `checksums.sha256`
- `version.json`
- harness payload files

The manifest is validated by `schemas/frozen-manifest.schema.json` before the archive is written.

## Verify

```bash
node packages/cli/dist/index.js frozen verify dist/x-harness-frozen.tar.gz --json
```

Verification reads the manifest, validates the manifest schema, and compares bundle payload hashes against both `manifest.json` and `checksums.sha256`.

## Import

Import verifies checksums before planning or writing files.

```bash
node packages/cli/dist/index.js import dist/x-harness-frozen.tar.gz --frozen --target ../target-repo --json
```

The default import mode is a dry run. It reports planned files and writes nothing.

To copy only files that do not already exist:

```bash
node packages/cli/dist/index.js import dist/x-harness-frozen.tar.gz --frozen --target ../target-repo --merge --json
```

To overwrite existing files:

```bash
node packages/cli/dist/index.js import dist/x-harness-frozen.tar.gz --frozen --target ../target-repo --force --json
```

Path traversal is rejected during verify/import, and unsafe archive paths are never written.

## Release Compatibility Gate

The release workflow installs the generated npm tarball and runs frozen transfer through the packed `xh` binary:

```bash
xh export --frozen --root "$PWD" --out x-harness-frozen.tar.gz --json
xh frozen verify x-harness-frozen.tar.gz --json
xh import x-harness-frozen.tar.gz --frozen --target frozen-target --merge --json
xh doctor --root frozen-target
```

This keeps frozen export/import compatibility tied to the same package artifact that would be published.
