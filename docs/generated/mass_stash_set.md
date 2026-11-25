---
id: mass_stash_set.md
slug: /cli/commands/mass_stash_set
title: Mass Stash Set
sidebar_label: Mass Stash Set
---
## mass stash set

Set a stash entry's value

### Synopsis

# Set Stash Entry

Stores a value under a key. A value of `-` reads the value from stdin and keeps every byte, including a trailing newline.

Each write replaces the entry's value, label, format, TTL, and flags.

## Usage

```bash
mass stash set <scope>.<key> <value> [flags]
```

## Flags

- `--label`: A human-readable name for the entry, up to 255 characters. It is not encrypted and is shown even for sensitive entries, so keep secrets out of it.
- `--format`: `text` (default) or `json`. A `json` value must be a JSON object.
- `--ttl`: Delete the entry this many seconds after the write
- `--immutable`: Keep the first value. Later writes fail.
- `--sensitive`: Hide the value from callers that cannot write the scope

## Examples

```bash
mass stash set "$DEPLOYMENT_ID.db/hostname" "$DB_HOST" --label "Database hostname"

# Run migrations once per deployment
mass stash set "$DEPLOYMENT_ID.migrations/ran" "true" --immutable

mass stash set "$DEPLOYMENT_ID.stats" "$(jq -n '{duration_s: 412, changed: 3}')" --format json

# Keep for one day
mass stash set "api-staging-db.ami-id" "ami-0abc" --ttl 86400

terraform output -raw kubeconfig | mass stash set "api-prod.kubeconfig" - --sensitive
```


```
mass stash set <scope>.<key> <value> [flags]
```

### Options

```
      --format string   How the value is read (text, json) (default "text")
  -h, --help            help for set
      --immutable       Keep the first value; later writes fail
      --label string    Human-readable name for the entry, up to 255 characters
      --sensitive       Hide the value from callers that cannot write the scope
      --ttl int         Delete the entry this many seconds after the write
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass stash](/cli/commands/mass_stash)	 - Store and read small values in a project, environment, component, instance, or deployment
