---
id: mass_environment_create.md
slug: /cli/commands/mass_environment_create
title: Mass Environment Create
sidebar_label: Mass Environment Create
---
## mass environment create

Create an environment

### Synopsis

# Create Environment

Creates a new environment in a project.

## Usage

```bash
mass environment create <slug> [flags]
```

## Flags

- `--name, -n`: Environment name (defaults to slug if not provided)

## Examples

```bash
# Create an environment "dbbundle-test" (project "dbbundle" is parsed from slug)
mass environment create dbbundle-test

# Create an environment with a custom name
mass environment create dbbundle-test --name "Database Test Environment"
```


```
mass environment create [slug] [flags]
```

### Options

```
  -h, --help          help for create
  -n, --name string   Environment name (defaults to slug if not provided)
```

### SEE ALSO

* [mass environment](/cli/commands/mass_environment)	 - Environment management
