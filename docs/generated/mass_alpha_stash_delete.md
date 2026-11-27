---
id: mass_alpha_stash_delete.md
slug: /cli/commands/mass_alpha_stash_delete
title: Mass Alpha Stash Delete
sidebar_label: Mass Alpha Stash Delete
---
## mass alpha stash delete

Delete a stash entry

### Synopsis

# Delete Stash Entry

Deletes an entry. You must type the key to confirm, unless you set `--force`. A different answer cancels the delete and exits 0, so scripts should use `--force`.

## Usage

```bash
mass alpha stash delete <scope>.<key> [flags]
```

## Flags

- `--force, -f`: Skip the confirmation prompt

## Examples

```bash
mass alpha stash delete "api-staging-db.ami-id"

mass alpha stash delete "env:api-db.config/v2" --force
```


```
mass alpha stash delete <scope>.<key> [flags]
```

### Options

```
  -f, --force   Skip the confirmation prompt
  -h, --help    help for delete
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass alpha stash](/cli/commands/mass_alpha_stash)	 - Store and read small values in a project, environment, component, instance, or deployment
