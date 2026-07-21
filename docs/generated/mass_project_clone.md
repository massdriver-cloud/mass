---
id: mass_project_clone.md
slug: /cli/commands/mass_project_clone
title: Mass Project Clone
sidebar_label: Mass Project Clone
---
## mass project clone

Clone a project's blueprint into a new project

### Synopsis

# Clone Project

Creates a new project by copying another project's blueprint. All components and links from the source are copied into the new project, which gets its own independent blueprint — subsequent changes to either project do not affect the other.

Environments are not cloned; create them separately with `mass environment create` (or `mass environment fork`).

## Usage

```bash
mass project clone <source-project> <new-id> [flags]
```

- `<source-project>` is the project ID/slug to clone from.
- `<new-id>` is the new project's short, memorable identifier (max 20 chars, lowercase alphanumeric). Immutable after creation.

## Flags

- `--name, -n`: New project name (defaults to `<new-id>` if not provided)
- `--description, -d`: Optional project description
- `--attributes, -a`: Custom attributes for ABAC (e.g. `-a team=ops,system=api`)

## Examples

```bash
# Clone the ecomm blueprint into a new project
mass project clone ecomm ecomm-copy

# Clone with a custom display name and attributes
mass project clone ecomm ecomm-eu -n "Ecomm (EU)" -a region=eu
```


```
mass project clone [source-project] [new-id] [flags]
```

### Examples

```
mass project clone ecomm ecomm-copy
```

### Options

```
  -a, --attributes stringToString   Custom attributes for ABAC (e.g. -a team=ops,system=api) (default [])
  -d, --description string          Optional project description
  -h, --help                        help for clone
  -n, --name string                 New project name (defaults to new-id if not provided)
```

### SEE ALSO

* [mass project](/cli/commands/mass_project)	 - Project management
