---
id: mass_package_destroy.md
slug: /cli/commands/mass_package_destroy
title: Mass Package Destroy
sidebar_label: Mass Package Destroy
---
## mass package destroy

Destroy (decommission) a package

### Synopsis

Destroy (decommission) a package. This will permanently delete the package and all its resources.

```
mass package destroy <project>-<env>-<manifest> [flags]
```

### Examples

```
mass package destroy api-prod-db --force
```

### Options

```
  -f, --force   Skip confirmation prompt
  -h, --help    help for destroy
```

### SEE ALSO

* [mass package](/cli/commands/mass_package)	 - Manage packages of IaC deployed in environments.
