---
id: mass_environment_get.md
slug: /cli/commands/mass_environment_get
title: Mass Environment Get
sidebar_label: Mass Environment Get
---
## mass environment get

Get an environment from Massdriver

### Synopsis

# Get Environment Details

Retrieves detailed information about a specific environment, including:
- Environment name and description
- Environment configurations
- Cost metrics
- Environment settings

## Usage

```bash
mass environment get <project-slug>-<environment-slug>
```

## Examples

```bash
# Get details for the "prod" environment in the "web" project
mass environment get web-prod
```


```
mass environment get [environment] [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass environment](/cli/commands/mass_environment)	 - Environment management
