---
id: mass_config_get.md
slug: /cli/commands/mass_config_get
title: Mass Config Get
sidebar_label: Mass Config Get
---
## mass config get

Show a profile's settings (defaults to the active profile)

### Synopsis

# Get a Profile

Shows the settings for one profile. With no argument, shows the active profile.

The API key is masked. Pass `--show-secrets` to print it in full.

## Usage

```bash
mass config get [profile]
```

## Examples

```bash
# Show the active profile
mass config get

# Show a specific profile
mass config get staging
```


```
mass config get [profile] [flags]
```

### Examples

```
mass config get staging
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text, json) (default "text")
      --show-secrets    Print the API key in full instead of masking it
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass config](/cli/commands/mass_config)	 - Manage CLI configuration profiles
