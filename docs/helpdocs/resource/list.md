# List Resources

Lists resources in your Massdriver organization.

Resources are either **imported** (registered manually, identified by UUID) or
**provisioned** (created by a deployment, identified by a friendly slug). Use the
filters below to narrow large result sets, and `--output json` for scripting.

## Usage

```bash
mass resource list [flags]
```

## Flags

- `-s, --search` — full-text search across the resource name
- `-t, --type` — filter by resource type id (e.g. `aws-iam-role`)
- `--origin` — filter by origin: `imported` or `provisioned` (empty matches both)
- `-e, --environment` — limit to provisioned resources in an environment
- `--sort` — sort field: `name` or `created_at`
- `--order` — sort order: `asc` or `desc`
- `-o, --output` — output format: `table` or `json`

## Examples

```bash
# List all resources
mass resource list

# Full-text search
mass resource list --search database

# Provisioned IAM roles, newest first
mass resource list --type aws-iam-role --origin provisioned --sort created_at --order desc

# Resources in a specific environment, as JSON
mass resource list --environment ecomm-prod -o json
```
