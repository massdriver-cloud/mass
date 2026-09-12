---
id: mass_deployment_logs.md
slug: /cli/commands/mass_deployment_logs
title: Mass Deployment Logs
sidebar_label: Mass Deployment Logs
---
## mass deployment logs

Stream the log output from a deployment

### Synopsis

# Get Deployment Logs

Prints the log output emitted by a deployment, oldest first. Each batch is a single worker flush; a batch's message may contain multiple newline-separated lines.

## Usage

```shell
mass deployment logs <deployment-id>
```

## Examples

```shell
mass deployment logs 12345678-1234-1234-1234-123456789012
```


```
mass deployment logs <deployment-id> [flags]
```

### Options

```
  -h, --help   help for logs
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass deployment](/cli/commands/mass_deployment)	 - Manage deployments
