---
id: mass_deployment_approve.md
slug: /cli/commands/mass_deployment_approve
title: Mass Deployment Approve
sidebar_label: Mass Deployment Approve
---
## mass deployment approve

Approve a proposed deployment, releasing it to run

### Synopsis

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


```
mass deployment approve <deployment-id> [flags]
```

### Examples

```
mass deployment approve 12345678-1234-1234-1234-123456789012
```

### Options

```
  -h, --help   help for approve
```

### SEE ALSO

* [mass deployment](/cli/commands/mass_deployment)	 - Manage deployments
