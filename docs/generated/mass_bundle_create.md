---
id: mass_bundle_create.md
slug: /cli/commands/mass_bundle_create
title: Mass Bundle Create
sidebar_label: Mass Bundle Create
---
## mass bundle create

Create a new bundle OCI repository in your organization's catalog

### Synopsis

# Create Bundle Repository

Creates an empty bundle OCI repository in your organization's catalog. The repository holds published versions of a bundle; create it before the first `mass bundle publish`.

## Usage

```bash
mass bundle create <name> [flags]
```

## Flags

- `-a, --attributes` — custom attributes for ABAC (repeat or comma-separate)

## Examples

```bash
# Create a repository for an Aurora Postgres bundle
mass bundle create aws-aurora-postgres

# Tag it with attributes policies and grants can match on
mass bundle create aws-aurora-postgres -a owner=data,service=database
```

## Notes

- This is `mass repository create --type bundle` with the type filled in.
- Share the repository with other projects using `mass repository grant create`.


```
mass bundle create <name> [flags]
```

### Options

```
  -a, --attributes stringToString   Custom attributes (e.g. -a owner=data,service=database) (default [])
  -h, --help                        help for create
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
