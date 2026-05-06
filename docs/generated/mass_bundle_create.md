---
id: mass_bundle_create.md
slug: /cli/commands/mass_bundle_create
title: Mass Bundle Create
sidebar_label: Mass Bundle Create
---
## mass bundle create

Create a new bundle OCI repository in your organization's catalog

```
mass bundle create <name> [flags]
```

### Examples

```
mass bundle create aws-aurora-postgres -a owner=data,service=database
```

### Options

```
  -a, --attributes stringToString   Custom attributes (e.g. -a owner=data,service=database) (default [])
  -h, --help                        help for create
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
