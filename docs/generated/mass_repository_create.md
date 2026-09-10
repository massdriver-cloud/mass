---
id: mass_repository_create.md
slug: /cli/commands/mass_repository_create
title: Mass Repository Create
sidebar_label: Mass Repository Create
---
## mass repository create

Create a new OCI repository

```
mass repository create <name> [flags]
```

### Options

```
  -a, --attributes stringToString   Custom attributes (e.g. -a owner=data,service=database) (default [])
  -h, --help                        help for create
  -t, --type string                 Artifact type (bundle, resource-type)
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass repository](/cli/commands/mass_repository)	 - Manage OCI repositories (bundles and resource types)
