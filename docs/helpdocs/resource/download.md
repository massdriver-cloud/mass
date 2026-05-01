# Download Resource

Downloads a resource in the specified format. The resource data is rendered according to the resource type's schema and returned in the requested format.

## Usage

```bash
mass resource download <resource-id> [flags]
```

Where `<resource-id>` can be:
- **UUID**: For imported resources (e.g., `12345678-1234-1234-1234-123456789012`)
- **Friendly slug**: For provisioned resources (e.g., `api-prod-database-connection`)

Friendly slug format: `PROJECT_SLUG-ENVIRONMENT_SLUG-MANIFEST_SLUG-BUNDLE_RESOURCE_FIELD_NAME`

## Examples

```bash
# Download resource in JSON format (default) using UUID
mass resource download 12345678-1234-1234-1234-123456789012

# Download resource using friendly slug
mass resource download api-prod-database-connection
mass resource download network-useast1-vpc-network

# Download resource in YAML format
mass resource download 12345678-1234-1234-1234-123456789012 --format yaml
mass resource download api-prod-grpcapi-host -f yaml

# Download resource in JSON format explicitly
mass resource download 12345678-1234-1234-1234-123456789012 --format json
```

## Options

- `--format, -f`: Download format (json, yaml, etc.). Defaults to json.

## Notes

- **Provisioned resources** (created by bundle deployments) can use either UUID or friendly slug
- **Imported resources** (created via CLI, API, or UI) must use UUID
- The available formats depend on the resource type's configuration
- Use `mass resource get` to see available formats for a specific resource
