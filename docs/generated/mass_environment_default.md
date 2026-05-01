---
id: mass_environment_default.md
slug: /cli/commands/mass_environment_default
title: Mass Environment Default
sidebar_label: Mass Environment Default
---
## mass environment default

Set an environment default connection

### Synopsis

# Set Environment Default

Sets a resource as the default connection for an environment.

## Usage

```bash
mass environment default [environment] [resource-id]
```

## Arguments

- `environment`: Environment ID or slug
- `resource-id`: Resource ID to set as default

## Examples

```bash
# Set a resource as default for an environment
mass env default api-prod abc123-def456
```


```
mass environment default [environment] [resource-id] [flags]
```

### Options

```
  -h, --help   help for default
```

### SEE ALSO

* [mass environment](/cli/commands/mass_environment)	 - Environment management
