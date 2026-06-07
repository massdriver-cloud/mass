---
id: mass_deployment_list.md
slug: /cli/commands/mass_deployment_list
title: Mass Deployment List
sidebar_label: Mass Deployment List
---
## mass deployment list

List deployments for an instance (most recent first)

### Synopsis

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


```
mass deployment list <instance-id> [flags]
```

### Examples

```
mass deployment list ecomm-prod-db --limit 25
```

### Options

```
      --action string   Filter by action (provision, decommission, plan)
      --bundle string   Filter by bundle version (name@version) or release channel (name@latest)
  -h, --help            help for list
  -n, --limit int       Maximum number of deployments to return (max 100) (default 10)
  -o, --output string   Output format (table, json) (default "table")
      --repo string     Filter by OCI repo name
      --status string   Filter by status (pending, approved, running, completed, failed, aborted, rejected, proposed)
```

### SEE ALSO

* [mass deployment](/cli/commands/mass_deployment)	 - Manage deployments
