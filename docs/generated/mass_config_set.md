---
id: mass_config_set.md
slug: /cli/commands/mass_config_set
title: Mass Config Set
sidebar_label: Mass Config Set
---
## mass config set

Update settings on an existing profile

### Synopsis

# Update a Profile

Updates settings on an existing profile. Only the values you pass are changed; everything else is left alone. Pass an empty string to remove a setting.

Credentials are verified against the Massdriver API before the change is written. Pass `--no-verify` to skip that check.

## Usage

```bash
mass config set <profile>
```

## Examples

```bash
# Rotate the API key
mass config set staging --api-key mds_yyy

# Point a profile at a self-hosted installation
mass config set staging --url https://api.massdriver.example.com

# Set where `mass bundle new` looks for templates
mass config set staging --templates-path ~/massdriver/templates

# Clear the templates path
mass config set staging --templates-path ""
```


```
mass config set [profile] [flags]
```

### Examples

```
mass config set staging --url https://api.massdriver.cloud
```

### Options

```
      --api-key string          API key or personal access token
  -h, --help                    help for set
      --no-verify               Skip checking the credentials against the Massdriver API
      --org string              Organization abbreviation
      --templates-path string   Directory containing bundle templates
      --url string              Massdriver API URL (defaults to https://api.massdriver.cloud)
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass config](/cli/commands/mass_config)	 - Manage CLI configuration profiles
