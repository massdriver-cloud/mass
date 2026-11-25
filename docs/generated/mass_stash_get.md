---
id: mass_stash_get.md
slug: /cli/commands/mass_stash_get
title: Mass Stash Get
sidebar_label: Mass Stash Get
---
## mass stash get

Print a stash entry's value

### Synopsis

# Get Stash Entry

Prints an entry's value exactly as it is stored, with no trailing newline.

A sensitive value is shown only to callers that can write its scope. Other callers get an error.

## Usage

```bash
mass stash get <scope>.<key> [flags]
```

## Flags

- `--output, -o`: Output format, `text` (default) or `json`. `json` prints the whole entry: its value, format, flags, and expiry.

## Examples

```bash
DB_HOST=$(mass stash get "$DEPLOYMENT_ID.db/hostname")

mass stash get "$DEPLOYMENT_ID.stats" -o json | jq .data.changed
```


```
mass stash get <scope>.<key> [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text, json) (default "text")
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass stash](/cli/commands/mass_stash)	 - Store and read small values in a project, environment, component, instance, or deployment
