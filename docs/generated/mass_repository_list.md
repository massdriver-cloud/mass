---
id: mass_repository_list.md
slug: /cli/commands/mass_repository_list
title: Mass Repository List
sidebar_label: Mass Repository List
---
## mass repository list

List OCI repositories

```
mass repository list [flags]
```

### Options

```
  -h, --help            help for list
  -n, --name string     Filter by exact repository name
      --order string    Sort order (asc, desc) (default "asc")
  -o, --output string   Output format (table, json) (default "table")
      --prefix string   Filter by repository name prefix
  -s, --search string   Full-text search across name, readme, and changelog
      --sort string     Sort field (name, created_at)
  -t, --type string     Filter by artifact type (bundle)
```

### SEE ALSO

* [mass repository](/cli/commands/mass_repository)	 - Manage OCI repositories (bundles and, in future, resource types and provisioners)
