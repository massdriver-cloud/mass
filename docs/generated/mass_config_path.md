---
id: mass_config_path.md
slug: /cli/commands/mass_config_path
title: Mass Config Path
sidebar_label: Mass Config Path
---
## mass config path

Print the path to the configuration file

### Synopsis

# Print the Configuration File Path

Prints where the CLI reads its configuration from, so you can open or back up the file directly.

The location is `$XDG_CONFIG_HOME/massdriver/config.yaml` when `XDG_CONFIG_HOME` is set, and `~/.config/massdriver/config.yaml` otherwise.

## Usage

```bash
mass config path
```

## Examples

```bash
# Print the path
mass config path

# Open the file in your editor
$EDITOR "$(mass config path)"
```


```
mass config path [flags]
```

### Examples

```
mass config path
```

### Options

```
  -h, --help   help for path
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass config](/cli/commands/mass_config)	 - Manage CLI configuration profiles
