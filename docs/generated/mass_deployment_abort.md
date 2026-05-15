---
id: mass_deployment_abort.md
slug: /cli/commands/mass_deployment_abort
title: Mass Deployment Abort
sidebar_label: Mass Deployment Abort
---
## mass deployment abort

Abort a pending, approved, or running deployment

### Synopsis

# Abort a Deployment

Cancels a `PENDING`, `APPROVED`, or `RUNNING` deployment. The deployment transitions to `ABORTED`.

A running deployment aborted mid-flight leaves any partial infrastructure changes the provisioner had applied in place — the instance's state is left as it was at the moment of abort.

To discard a `PROPOSED` deployment instead, use the `reject` flow.

## Usage

```shell
mass deployment abort <deployment-id> [--force]
```

## Flags

- `--force, -f`: Skip the confirmation prompt.

## Examples

```shell
mass deployment abort 12345678-1234-1234-1234-123456789012
mass deployment abort 12345678-1234-1234-1234-123456789012 --force
```


```
mass deployment abort <deployment-id> [flags]
```

### Examples

```
mass deployment abort 12345678-1234-1234-1234-123456789012 --force
```

### Options

```
  -f, --force   Skip confirmation prompt
  -h, --help    help for abort
```

### SEE ALSO

* [mass deployment](/cli/commands/mass_deployment)	 - Manage deployments
