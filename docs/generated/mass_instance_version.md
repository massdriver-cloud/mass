---
id: mass_instance_version.md
slug: /cli/commands/mass_instance_version
title: Mass Instance Version
sidebar_label: Mass Instance Version
---
## mass instance version

Set instance version

### Synopsis

# Set instance version

Set the version for an instance in Massdriver.

## Examples

Set the version for an instance using the `slug@version` format:

```shell
mass instance version api-prod-db@latest
```

The `slug` can be found in the instance info panel. The instance slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`.

## Version Format

The version can be:
- A semantic version (e.g., `1.2.3`)
- A version constraint (e.g., `~1.2`, `~1`)
- A release channel (e.g., `latest`)


```
mass instance version <instance-id>@<version> [flags]
```

### Examples

```
mass instance version api-prod-db@latest
```

### Options

```
  -h, --help   help for version
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
