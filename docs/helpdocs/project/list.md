# List Projects

Lists all Massdriver projects in your organization.

## Usage

```bash
mass project list [flags]
```

## Flags

- `--output, -o`: Output format, `table` (default) or `json`
- `--name`: Filter to projects whose name exactly matches the given value
- `--search`: Free-text search across the project's name and description (matches whole words anywhere; results are ranked by relevance)

## Examples

```bash
# List all projects
mass project list

# Filter to a project by exact name
mass project list --name "Ecommerce"

# Free-text search across name and description
mass project list --search ecomm

# JSON output
mass project list -o json
```
