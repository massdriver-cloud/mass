---
id: mass_instance_orphan.md
slug: /cli/commands/mass_instance_orphan
title: Mass Instance Orphan
sidebar_label: Mass Instance Orphan
---
## mass instance orphan

Orphan an instance (reset to INITIALIZED, clearing state locks)

### Synopsis

# Orphan an Instance

Resets an instance to `INITIALIZED`, clearing all of its Terraform/OpenTofu state locks. This is a break-glass operation for instances that are permanently stuck — active `RUNNING`, `PENDING`, and `APPROVED` deployments are bulk-aborted so a late worker callback cannot walk the instance status back to `PROVISIONED`.

By default, the remote state files are preserved so the next deployment can re-attach to existing infrastructure. Pass `--delete-state` to also permanently delete the state files.

## Usage

```shell
mass instance orphan <project>-<env>-<manifest> [--delete-state] [--force]
```

## Flags

- `--force, -f`: Skip the confirmation prompt.
- `--delete-state`: Also permanently delete the instance's Terraform/OpenTofu state files. The next deployment will provision from scratch and may duplicate any resources tracked by the prior state. **Irreversible.**

## Examples

```shell
mass instance orphan api-prod-db
mass instance orphan api-prod-db --force
mass instance orphan api-prod-db --delete-state
```


```
mass instance orphan <project>-<env>-<manifest> [flags]
```

### Examples

```
mass instance orphan api-prod-db --force
```

### Options

```
      --delete-state   Also delete the remote Terraform/OpenTofu state files (irreversible)
  -f, --force          Skip confirmation prompt
  -h, --help           help for orphan
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
