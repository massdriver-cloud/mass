---
id: mass_alpha_stash_list.md
slug: /cli/commands/mass_alpha_stash_list
title: Mass Alpha Stash List
sidebar_label: Mass Alpha Stash List
---
## mass alpha stash list

List the stash entries in a scope

### Synopsis

# List Stash Entries

Lists the entries in one scope. Sensitive values show as `(sensitive)` unless you can write the scope.

## Usage

```bash
mass alpha stash list <scope> [flags]
```

## Flags

- `--output, -o`: Output format, `table` (default) or `json`

## Examples

```bash
mass alpha stash list api-prod

mass alpha stash list "$DEPLOYMENT_ID" -o json | jq -r '.[].key'
```


```
mass alpha stash list <scope> [flags]
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

* [mass alpha stash](/cli/commands/mass_alpha_stash)	 - Store and read small values in a project, environment, component, instance, or deployment
