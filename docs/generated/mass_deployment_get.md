---
id: mass_deployment_get.md
slug: /cli/commands/mass_deployment_get
title: Mass Deployment Get
sidebar_label: Mass Deployment Get
---
## mass deployment get

Get a deployment by ID

### Synopsis

# Get a Deployment

Retrieves a single deployment by its UUID, including status, action, version, and timing.

## Usage

```shell
mass deployment get <deployment-id> [--output text|json]
```

## Examples

```shell
mass deployment get 12345678-1234-1234-1234-123456789012
mass deployment get 12345678-1234-1234-1234-123456789012 --output json
```


```
mass deployment get <deployment-id> [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass deployment](/cli/commands/mass_deployment)	 - Manage deployments
