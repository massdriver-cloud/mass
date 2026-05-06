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
      --follow              Stream the deployment's logs to stdout until it completes
  -f, --force               Skip confirmation prompt
  -h, --help                help for destroy
  -m, --message string      Add a message when decommissioning
  -p, --params string       Path to params json, tfvars or yaml file. Use '-' to read from stdin. When provided, the full configuration is replaced. Supports bash interpolation.
  -P, --patch stringArray   Patch the last deployed configuration using a JQ expression. Can be specified multiple times.
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
