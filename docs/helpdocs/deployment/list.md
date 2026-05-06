# List Deployments for an Instance

Lists deployments for the given instance, most recent first. By default returns the 10 most recent. Use `--limit` to return more (capped at 100 by the server).

## Usage

```shell
mass deployment list <instance-id> [--limit N]
```

## Examples

```shell
# Ten most recent deployments for the ecomm-prod-db instance
mass deployment list ecomm-prod-db

# Last 50
mass deployment list ecomm-prod-db --limit 50
```
