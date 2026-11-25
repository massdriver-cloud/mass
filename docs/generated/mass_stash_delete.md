---
id: mass_stash_delete.md
slug: /cli/commands/mass_stash_delete
title: Mass Stash Delete
sidebar_label: Mass Stash Delete
---
## mass stash delete

Delete a stash entry

### Synopsis

# Delete Stash Entry

Deletes an entry. You must type the key to confirm, unless you set `--force`.

## Usage

```bash
mass stash delete <scope>.<key> [flags]
```

## Flags

- `--force, -f`: Skip the confirmation prompt

## Examples

```bash
mass stash delete "api-staging-db.ami-id"

mass stash delete "env:api-db.config/v2" --force
```


```
mass stash delete <scope>.<key> [flags]
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

* [mass stash](/cli/commands/mass_stash)	 - Store and read small values in a project, environment, component, instance, or deployment
