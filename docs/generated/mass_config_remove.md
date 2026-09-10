---
id: mass_config_remove.md
slug: /cli/commands/mass_config_remove
title: Mass Config Remove
sidebar_label: Mass Config Remove
---
## mass config remove

Remove a profile

### Synopsis

# Remove a Profile

Removes a profile from the configuration file. You are asked to confirm by typing the profile name unless `--force` is passed.

If the removed profile was the active one, `current_profile` is cleared so later commands fall back to the `default` profile instead of failing on a name that no longer exists.

## Usage

```bash
mass config remove <profile>
```

## Examples

```bash
# Remove a profile
mass config remove staging

# Remove without the confirmation prompt
mass config remove staging --force
```


```
mass config remove [profile] [flags]
```

### Examples

```
mass config remove staging
```

### Options

```
  -f, --force   Skip confirmation prompt
  -h, --help    help for remove
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass config](/cli/commands/mass_config)	 - Manage CLI configuration profiles
