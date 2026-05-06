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


```
mass bundle list [flags]
```

### Options

```
  -h, --help            help for list
  -n, --name string     Filter by exact bundle name
      --order string    Sort order (asc, desc) (default "asc")
  -o, --output string   Output format (table, json) (default "table")
  -s, --search string   Search bundles by name, readme, and changelog
      --sort string     Sort field (name, created_at). Defaults to name, or relevance when using --search
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
