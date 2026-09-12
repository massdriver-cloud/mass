---
id: mass_config_add.md
slug: /cli/commands/mass_config_add
title: Mass Config Add
sidebar_label: Mass Config Add
---
## mass config add

Add a profile

### Synopsis

# Add a Profile

Adds a profile to the configuration file. When run in a terminal, you are prompted for any required value you did not pass as a flag.

The credentials are checked against the Massdriver API before the profile is written, so a mistyped key or organization fails here rather than on your next command. Pass `--no-verify` to skip that check.

The first profile you add becomes the active profile.

## Usage

```bash
mass config add <profile>
```

## Examples

```bash
# Add a profile interactively
mass config add staging

# Add a profile non-interactively
mass config add staging --org acme --api-key mds_xxx

# Add a profile pointing at a self-hosted installation
mass config add onprem --org acme --api-key mds_xxx --url https://api.massdriver.example.com

# Add a profile and make it active
mass config add staging --org acme --api-key mds_xxx --use
```


```
mass config add [profile] [flags]
```

### Examples

```
mass config add staging --org acme --api-key mds_xxx
```

### Options

```
      --api-key string          API key or personal access token
  -f, --force                   Overwrite the profile if it already exists
  -h, --help                    help for add
      --no-verify               Skip checking the credentials against the Massdriver API
      --org string              Organization abbreviation
      --templates-path string   Directory containing bundle templates
      --url string              Massdriver API URL (defaults to https://api.massdriver.cloud)
      --use                     Make this the active profile after adding it
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass config](/cli/commands/mass_config)	 - Manage CLI configuration profiles
