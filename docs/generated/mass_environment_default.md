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

Sets an artifact as the default connection for an environment.

## Usage

```bash
mass environment default [environment] [artifact-id]
```

## Arguments

- `environment`: Environment ID or slug
- `artifact-id`: Artifact ID to set as default

## Examples

```bash
# Set an artifact as default for an environment
mass env default api-prod abc123-def456
```


```
mass environment default [environment] [artifact-id] [flags]
```

### Options

```
  -h, --help   help for default
```

### SEE ALSO

* [mass environment](/cli/commands/mass_environment)	 - Environment management
