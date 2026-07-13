# Reject a Deployment

Discards a `PROPOSED` deployment permanently. The deployment transitions to `REJECTED`, which is terminal — rejected deployments never run.

Only valid for deployments currently in `PROPOSED` status. To cancel a `PENDING`, `APPROVED`, or `RUNNING` deployment instead, use the `abort` flow.

## Usage

```shell
mass deployment reject <deployment-id>
```

## Examples

```shell
mass deployment reject 12345678-1234-1234-1234-123456789012
```
