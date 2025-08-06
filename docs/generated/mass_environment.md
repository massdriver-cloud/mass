---
id: mass_environment.md
slug: /cli/commands/mass_environment
title: Mass Environment
sidebar_label: Mass Environment
---
## mass environment

Environment management

### Synopsis

# Manage Environments

[Environments](https://docs.massdriver.cloud/concepts/environments) are the workspaces that bundles will be deployed to.

Environments can be modeled by application stage (production, staging, development), by region (prod-usw, prod-eu), and even ephemerally per developer (alice-dev, bob-dev).

## Commands

- `export`: Export an environment to local filesystem
- `get`: Retrieve environment details and configuration
- `list`: List all environments in a project


### Options

```
  -h, --help   help for environment
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass environment export](/cli/commands/mass_environment_export)	 - Export an environment from Massdriver
* [mass environment get](/cli/commands/mass_environment_get)	 - Get an environment from Massdriver
* [mass environment list](/cli/commands/mass_environment_list)	 - List environments
