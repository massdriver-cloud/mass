---
id: mass_config_list.md
slug: /cli/commands/mass_config_list
title: Mass Config List
sidebar_label: Mass Config List
---
## mass config list

List configured profiles

### Synopsis

# List Profiles

Lists every profile in the configuration file. The active profile is marked with `*`.

API keys are masked. Pass `--show-secrets` to print them in full.

## Usage

```bash
mass config list
```

## Examples

```bash
# List profiles
mass config list

# List profiles as JSON
mass config list --output json
```


```
mass config list [flags]
```

### Examples

```
mass config list
```

### Options

```
  -h, --help            help for list
  -o, --output string   Output format (table, json) (default "table")
      --show-secrets    Print API keys in full instead of masking them
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass config](/cli/commands/mass_config)	 - Manage CLI configuration profiles
