---
id: mass_component_update.md
slug: /cli/commands/mass_component_update
title: Mass Component Update
sidebar_label: Mass Component Update
---
## mass component update

Update a component's name, description, or attributes

```
mass component update <component-id> [flags]
```

### Examples

```
mass component update ecomm-db --name "Primary DB" -a priority=high
```

### Options

```
  -a, --attributes stringToString   Replacement custom attributes (e.g. -a priority=high,cost-center=engineering) (default [])
  -d, --description string          New description
  -h, --help                        help for update
  -n, --name string                 New display name
```

### SEE ALSO

* [mass component](/cli/commands/mass_component)	 - Manage components in a project's blueprint
