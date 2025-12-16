---
id: mass_artifact_get.md
slug: /cli/commands/mass_artifact_get
title: Mass Artifact Get
sidebar_label: Mass Artifact Get
---
## mass artifact get

Get an artifact from Massdriver

### Synopsis

# Get Artifact

Retrieves detailed information about a specific artifact, including:
- Artifact ID, name, and type
- Artifact definition details
- Package information (if provisioned)
- Specs and metadata
- Available download formats

## Usage

```bash
mass artifact get <artifact-id> [flags]
```

Where `<artifact-id>` can be:
- **UUID**: For imported artifacts (e.g., `12345678-1234-1234-1234-123456789012`)
- **Friendly slug**: For provisioned artifacts (e.g., `api-prod-database-connection`)

Friendly slug format: `PROJECT_SLUG-ENVIRONMENT_SLUG-MANIFEST_SLUG-BUNDLE_ARTIFACT_FIELD_NAME`

## Examples

```bash
# Get artifact details in text format (default) using UUID
mass artifact get 12345678-1234-1234-1234-123456789012

# Get artifact details using friendly slug
mass artifact get api-prod-database-connection
mass artifact get network-useast1-vpc-network

# Get artifact details in JSON format
mass artifact get 12345678-1234-1234-1234-123456789012 --output json
mass artifact get api-prod-grpcapi-host -o json
```

## Options

- `--output, -o`: Output format (text or json). Defaults to text.

## Notes

- **Provisioned artifacts** (created by bundle deployments) can use either UUID or friendly slug
- **Imported artifacts** (created via CLI, API, or UI) must use UUID


```
mass artifact get [artifact-id] [flags]
```

### Examples

```
  # Get artifact using UUID (imported artifacts)
  mass artifact get 12345678-1234-1234-1234-123456789012

  # Get artifact using friendly slug (provisioned artifacts)
  mass artifact get api-prod-database-connection
  mass artifact get api-prod-grpcapi-host -o json
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass artifact](/cli/commands/mass_artifact)	 - Manage artifacts
