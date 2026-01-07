---
id: mass_package_reset.md
slug: /cli/commands/mass_package_reset
title: Mass Package Reset
sidebar_label: Mass Package Reset
---
## mass package reset

Reset package status to 'Initialized'

### Synopsis

# Reset Package Status

This command allows you to reset a package status back to 'Initialized'. This should only be used when a package is in an unrecoverable state - common situations include a package stuck in 'Pending' due to deployment issues, or a package that cannot be successfully decommissioned due to deployment failures.

## Examples

You can reset the package using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`

Reset and delete the deployment history:

```shell
mass package reset ecomm-prod-vpc
```

```
mass package reset <project>-<env>-<manifest> [flags]
```

### Examples

```
mass package reset api-prod-db
```

### Options

```
  -h, --help   help for reset
```

### SEE ALSO

* [mass package](/cli/commands/mass_package)	 - Manage packages of IaC deployed in environments.
