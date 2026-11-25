---
id: mass_stash_list.md
slug: /cli/commands/mass_stash_list
title: Mass Stash List
sidebar_label: Mass Stash List
---
## mass stash list

List the stash entries in a scope

### Synopsis

# List Stash Entries

Lists the entries in one scope. Sensitive values show as `(sensitive)` unless you can write the scope.

## Usage

```bash
mass stash list <scope> [flags]
```

## Flags

- `--output, -o`: Output format, `table` (default) or `json`

## Examples

```bash
mass stash list api-prod

mass stash list "$DEPLOYMENT_ID" -o json | jq -r '.[].key'
```


```
mass stash list <scope> [flags]
```

### Options

```
  -h, --help            help for list
  -o, --output string   Output format (table, json) (default "table")
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass stash](/cli/commands/mass_stash)	 - Store and read small values in a project, environment, component, instance, or deployment
