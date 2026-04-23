---
id: mass_resource_get.md
slug: /cli/commands/mass_resource_get
title: Mass Resource Get
sidebar_label: Mass Resource Get
---
## mass resource get

Get an resource from Massdriver

### Synopsis

# Get Resource

Retrieves detailed information about a specific resource, including:
- Resource ID, name, and type
- Resource type details
- Instance information (if provisioned)
- Payload and metadata
- Available download formats

## Usage

```bash
mass resource get <resource-id> [flags]
```

Where `<resource-id>` can be:
- **UUID**: For imported resources (e.g., `12345678-1234-1234-1234-123456789012`)
- **Friendly slug**: For provisioned resources (e.g., `api-prod-database-connection`)

Friendly slug format: `PROJECT_SLUG-ENVIRONMENT_SLUG-MANIFEST_SLUG-BUNDLE_RESOURCE_FIELD_NAME`

## Examples

```bash
# Get resource details in text format (default) using UUID
mass resource get 12345678-1234-1234-1234-123456789012

# Get resource details using friendly slug
mass resource get api-prod-database-connection
mass resource get network-useast1-vpc-network

# Get resource details in JSON format
mass resource get 12345678-1234-1234-1234-123456789012 --output json
mass resource get api-prod-grpcapi-host -o json
```

## Options

- `--output, -o`: Output format (text or json). Defaults to text.

## Notes

- **Provisioned resources** (created by bundle deployments) can use either UUID or friendly slug
- **Imported resources** (created via CLI, API, or UI) must use UUID


```
mass resource get [resource-id] [flags]
```

### Examples

```
  # Get resource using UUID (imported resources)
  mass resource get 12345678-1234-1234-1234-123456789012

  # Get resource using friendly slug (provisioned resources)
  mass resource get api-prod-database-connection
  mass resource get api-prod-grpcapi-host -o json
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass resource](/cli/commands/mass_resource)	 - Manage resources
