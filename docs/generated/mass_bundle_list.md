---
id: mass_bundle_list.md
slug: /cli/commands/mass_bundle_list
title: Mass Bundle List
sidebar_label: Mass Bundle List
---
## mass bundle list

List bundles in your organization

### Synopsis

# List bundles in your organization

List all bundle repositories with optional search, sort, and limit.

## Usage

```bash
mass bundle list [flags]
```

## Flags

- `--search, -s`: Search bundles using Google-style operators (AND, OR, -, quotes)
- `--sort`: Sort field (name, created_at). Defaults to "name"
- `--order`: Sort order (asc, desc). Defaults to "asc"
- `--limit, -l`: Maximum number of results to return
- `--output, -o`: Output format (table or json). Defaults to "table"

## Examples

```bash
# List all bundles
mass bundle list

# Search for postgres bundles
mass bundle list --search postgres

# Search with operators
mass bundle list --search "postgres AND aws -aurora"

# Sort by creation date, newest first
mass bundle list --sort created_at --order desc

# Limit results
mass bundle list --limit 10

# Output as JSON
mass bundle list -o json
```


```
mass bundle list [flags]
```

### Options

```
  -h, --help            help for list
  -l, --limit int       Maximum number of results to return
      --order string    Sort order (asc, desc) (default "asc")
  -o, --output string   Output format (table, json) (default "table")
  -s, --search string   Search bundles (supports AND, OR, -, quotes)
      --sort string     Sort field (name, created_at) (default "name")
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
