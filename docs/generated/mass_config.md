---
id: mass_config.md
slug: /cli/commands/mass_config
title: Mass Config
sidebar_label: Mass Config
---
## mass config

Manage CLI configuration profiles

### Synopsis

# Manage Configuration Profiles

A profile holds the organization, credential, and API URL used to talk to Massdriver. Profiles live in `~/.config/massdriver/config.yaml` (or `$XDG_CONFIG_HOME/massdriver/config.yaml`), and these commands read and write that file for you.

Use separate profiles to work across organizations or against a self-hosted installation without re-editing the file each time.

## Choosing a profile

A profile is selected in this order, highest precedence first:

1. The `--profile` flag
2. The `MASSDRIVER_PROFILE` environment variable
3. The `current_profile` key in the config file, set by `mass config use`
4. The profile named `default`

The first three name a profile explicitly, so a name that isn't in the file is an error rather than a silent fallback. The `default` fallback may be absent — that is how credentials supplied entirely through environment variables resolve.

## Getting started

```bash
mass config add default --org acme --api-key mds_xxx
mass config list
```


### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass config add](/cli/commands/mass_config_add)	 - Add a profile
* [mass config get](/cli/commands/mass_config_get)	 - Show a profile's settings (defaults to the active profile)
* [mass config list](/cli/commands/mass_config_list)	 - List configured profiles
* [mass config path](/cli/commands/mass_config_path)	 - Print the path to the configuration file
* [mass config remove](/cli/commands/mass_config_remove)	 - Remove a profile
* [mass config set](/cli/commands/mass_config_set)	 - Update settings on an existing profile
* [mass config use](/cli/commands/mass_config_use)	 - Set the active profile
