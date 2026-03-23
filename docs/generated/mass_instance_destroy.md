---
id: mass_instance_destroy.md
slug: /cli/commands/mass_instance_destroy
title: Mass Instance Destroy
sidebar_label: Mass Instance Destroy
---
## mass instance destroy

Destroy (decommission) an instance

### Synopsis

Destroy (decommission) an instance. This will permanently delete the instance and all its resources.

```
mass instance destroy <project>-<env>-<manifest> [flags]
```

### Examples

```
mass instance destroy api-prod-db --force
```

### Options

```
  -f, --force   Skip confirmation prompt
  -h, --help    help for destroy
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
