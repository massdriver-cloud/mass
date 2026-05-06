# Set Environment Default

Sets a resource as the default connection for an environment.

## Usage

```bash
mass environment default [environment] [resource-id]
```

## Arguments

- `environment`: Environment ID or slug
- `resource-id`: Resource ID to set as default

## Examples

```bash
# Set a resource as default for an environment
mass env default api-prod abc123-def456
```
