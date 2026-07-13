# Approve a Deployment

Releases a `PROPOSED` deployment into the run queue. The deployment transitions to `APPROVED` and runs as soon as nothing else is running on the instance.

Only valid for deployments currently in `PROPOSED` status. Create a proposal with `mass instance deploy <instance-id> --propose`.

## Usage

```shell
mass deployment approve <deployment-id>
```

## Examples

```shell
mass deployment approve 12345678-1234-1234-1234-123456789012
```
