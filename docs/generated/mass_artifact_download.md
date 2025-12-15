---
id: mass_artifact_download.md
slug: /cli/commands/mass_artifact_download
title: Mass Artifact Download
sidebar_label: Mass Artifact Download
---
## mass artifact download

Download an artifact in the specified format

### Synopsis

# Download Artifact

Downloads an artifact in the specified format. The artifact data is rendered according to the artifact definition's schema and returned in the requested format.

## Usage

```bash
mass artifact download <artifact-id> [flags]
```

Where `<artifact-id>` can be:
- **UUID**: For imported artifacts (e.g., `12345678-1234-1234-1234-123456789012`)
- **Friendly slug**: For provisioned artifacts (e.g., `api-prod-database-connection`)

Friendly slug format: `PROJECT_SLUG-ENVIRONMENT_SLUG-MANIFEST_SLUG-BUNDLE_ARTIFACT_FIELD_NAME`

## Examples

```bash
# Download artifact in JSON format (default) using UUID
mass artifact download 12345678-1234-1234-1234-123456789012

# Download artifact using friendly slug
mass artifact download api-prod-database-connection
mass artifact download network-useast1-vpc-network

# Download artifact in YAML format
mass artifact download 12345678-1234-1234-1234-123456789012 --format yaml
mass artifact download api-prod-grpcapi-host -f yaml

# Download artifact in JSON format explicitly
mass artifact download 12345678-1234-1234-1234-123456789012 --format json
```

## Options

- `--format, -f`: Download format (json, yaml, etc.). Defaults to json.

## Notes

- **Provisioned artifacts** (created by bundle deployments) can use either UUID or friendly slug
- **Imported artifacts** (created via CLI, API, or UI) must use UUID
- The available formats depend on the artifact definition's configuration
- Use `mass artifact get` to see available formats for a specific artifact


```
mass artifact download [artifact-id] [flags]
```

### Examples

```
  # Download artifact using UUID (imported artifacts)
  mass artifact download 12345678-1234-1234-1234-123456789012

  # Download artifact using friendly slug (provisioned artifacts)
  mass artifact download api-prod-database-connection
  mass artifact download network-useast1-vpc-network -f yaml
```

### Options

```
  -f, --format string   Download format (json, yaml, etc.) (default "json")
  -h, --help            help for download
```

### SEE ALSO

* [mass artifact](/cli/commands/mass_artifact)	 - Manage artifacts
