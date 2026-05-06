---
id: mass_component_unlink.md
slug: /cli/commands/mass_component_unlink
title: Mass Component Unlink
sidebar_label: Mass Component Unlink
---
## mass component unlink

Remove a link between two components

### Synopsis

# Unlink Two Components

Removes a design-time link between two components, identified by the `fromComponent.fromField → toComponent.toField` wiring. Existing connections in deployed environments are not affected until the next deployment.

## Usage

```shell
mass component unlink <from-component>.<from-field> <to-component>.<to-field>
```

## Examples

```shell
mass component unlink ecomm-db.authentication ecomm-app.database
```


```
mass component unlink <from-component>.<from-field> <to-component>.<to-field> [flags]
```

### Examples

```
mass component unlink ecomm-db.authentication ecomm-app.database
```

### Options

```
  -h, --help   help for unlink
```

### SEE ALSO

* [mass component](/cli/commands/mass_component)	 - Manage components in a project's blueprint
