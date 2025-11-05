# Set Environment Default

Sets an artifact as the default connection for an environment.

## Usage

```bash
mass environment default [environment] [artifact-id]
```

## Arguments

- `environment`: Environment ID or slug
- `artifact-id`: Artifact ID to set as default

## Examples

```bash
# Set an artifact as default for an environment
mass env default api-prod abc123-def456
```
