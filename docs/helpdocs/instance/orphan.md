# Orphan an Instance

Resets an instance to `INITIALIZED`, clearing all of its Terraform/OpenTofu state locks. This is a break-glass operation for instances that are permanently stuck, such as instances in a `FAILED` state that cannot be successfully
provisioned or decommissioned. Active `RUNNING`, `PENDING`, and `APPROVED` deployments are bulk-aborted so a late worker will not retry deployments.

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
