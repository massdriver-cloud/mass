---
id: mass_component_remove.md
slug: /cli/commands/mass_component_remove
title: Mass Component Remove
sidebar_label: Mass Component Remove
---
## mass component remove

Remove a component from a project's blueprint

### Synopsis

# Remove a Component from a Project

Removes a component from a project's blueprint, along with all its links. Any deployed instances of this component must be decommissioned first.

## Usage

```shell
mass component remove <component-id>
```

The component ID is in the `<project-id>-<component-id>` format.

## Examples

```shell
mass component remove ecomm-db
```


```
mass component remove <component-id> [flags]
```

### Examples

```
mass component remove ecomm-db
```

### Options

```
  -h, --help   help for remove
```

### SEE ALSO

* [mass component](/cli/commands/mass_component)	 - Manage components in a project's blueprint
