# List bundles in your organization

List all bundle repositories with optional search, filter, and sort.

## Usage

```bash
mass bundle list [flags]
```

## Flags

- `--search, -s`: Search bundles by name, readme, and changelog. Results ranked by relevance by default
- `--name, -n`: Filter by exact bundle name
- `--sort`: Sort field (name, created_at). Defaults to name, or relevance when using --search
- `--order`: Sort order (asc, desc). Defaults to "asc"
- `--output, -o`: Output format (table or json). Defaults to "table"

## Examples

```bash
# List all bundles
mass bundle list

# Search for postgres bundles
mass bundle list --search postgres

# Filter by exact name
mass bundle list --name aws-vpc

# Sort by creation date, newest first
mass bundle list --sort created_at --order desc

# Output as JSON
mass bundle list -o json
```
