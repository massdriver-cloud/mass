# Abort a Deployment

Cancels a `PENDING`, `APPROVED`, or `RUNNING` deployment. The deployment transitions to `ABORTED`.

A running deployment aborted mid-flight will not cancel or halt the
running provisioner. It only transitions the state of the Massdriver
deployment to `ABORTED`. Any partial infrastructure changes the
provisioner had applied will remain in place — the instance's state is
left as it was at the moment of abort.

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
