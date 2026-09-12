# Update Component

Updates a component's display name, description, or custom attributes.

Attributes are replaced wholesale rather than merged, so pass every attribute you want to keep.

## Usage

```bash
mass component update <component-id> [flags]
```

## Flags

- `-n, --name` — new display name
- `-d, --description` — new description
- `-a, --attributes` — replacement custom attributes (repeat or comma-separate)

## Examples

```bash
# Rename a component
mass component update ecomm-db --name "Primary DB"

# Rename and retag it in one call
mass component update ecomm-db --name "Primary DB" -a priority=high

# Replace the full attribute set
mass component update ecomm-db -a priority=high,cost-center=engineering
```
