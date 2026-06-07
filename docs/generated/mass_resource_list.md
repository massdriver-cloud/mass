---
id: mass_resource_list.md
slug: /cli/commands/mass_resource_list
title: Mass Resource List
sidebar_label: Mass Resource List
---
## mass resource list

List resources

### Synopsis

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


```
mass resource list [flags]
```

### Examples

```
  # List all resources
  mass resource list

  # Search and filter
  mass resource list --search database
  mass resource list --type aws-iam-role --origin provisioned
  mass resource list --environment ecomm-prod -o json
```

### Options

```
  -e, --environment string   Limit to provisioned resources in an environment
  -h, --help                 help for list
      --order string         Sort order (asc, desc) (default "asc")
      --origin string        Filter by origin (imported, provisioned). Empty matches both
  -o, --output string        Output format (table, json) (default "table")
  -s, --search string        Full-text search across resource name
      --sort string          Sort field (name, created_at)
  -t, --type string          Filter by resource type id (e.g. aws-iam-role)
```

### SEE ALSO

* [mass resource](/cli/commands/mass_resource)	 - Manage resources
