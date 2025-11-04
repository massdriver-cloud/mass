---
id: mass_project_create.md
slug: /cli/commands/mass_project_create
title: Mass Project Create
sidebar_label: Mass Project Create
---
## mass project create

Create a project

### Synopsis

# Create Project

Creates a new project in Massdriver.

## Usage

```bash
mass project create <slug> [flags]
```

## Flags

- `--name, -n`: Project name (defaults to slug if not provided)

## Examples

```bash
# Create a project with slug "dbbundle"
mass project create dbbundle

# Create a project with a custom name
mass project create dbbundle --name "Database Bundle Project"
```


```
mass project create [slug] [flags]
```

### Options

```
  -h, --help          help for create
  -n, --name string   Project name (defaults to slug if not provided)
```

### SEE ALSO

* [mass project](/cli/commands/mass_project)	 - Project management
