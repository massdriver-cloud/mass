---
id: mass_config_use.md
slug: /cli/commands/mass_config_use
title: Mass Config Use
sidebar_label: Mass Config Use
---
## mass config use

Set the active profile

### Synopsis

# Set the Active Profile

Makes a profile the one every `mass` command uses. The choice is written to the `current_profile` key in the config file, so it persists across shells and sessions.

`MASSDRIVER_PROFILE` and the `--profile` flag both take precedence over this setting.

## Usage

```bash
mass config use <profile>
```

## Examples

```bash
# Switch to the staging profile
mass config use staging

# Run a single command against a different profile
mass project list --profile production
```


```
mass config use [profile] [flags]
```

### Examples

```
mass config use staging
```

### Options

```
  -h, --help   help for use
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass config](/cli/commands/mass_config)	 - Manage CLI configuration profiles
