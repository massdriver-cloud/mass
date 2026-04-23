---
id: mass_instance_reset.md
slug: /cli/commands/mass_instance_reset
title: Mass Instance Reset
sidebar_label: Mass Instance Reset
---
## mass instance reset

Reset instance status to 'Initialized'

### Synopsis

# Reset Instance Status

This command allows you to reset an instance status back to 'Initialized'. This should only be used when an instance is in an unrecoverable state - common situations include an instance stuck in 'Pending' due to deployment issues, or an instance that cannot be successfully decommissioned due to deployment failures.

## Examples

You can reset the instance using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The instance slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`

Reset and delete the deployment history:

```shell
mass instance reset ecomm-prod-vpc
```


```
mass instance reset <project>-<env>-<manifest> [flags]
```

### Examples

```
mass instance reset api-prod-db
```

### Options

```
  -f, --force   Skip confirmation prompt
  -h, --help    help for reset
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
