---
id: mass_component_update.md
slug: /cli/commands/mass_component_update
title: Mass Component Update
sidebar_label: Mass Component Update
---
## mass component update

Update a component's name, description, or attributes

### Synopsis

# Update Component

Updates a component's display name, description, or custom attributes.

Attributes are replaced wholesale rather than merged, so pass every attribute you want to keep.

## Usage

```bash
mass component update <component-id> [flags]
```

## Flags

- `-n, --name` — new display name
- `-d, --description` — new description
- `-a, --attributes` — replacement custom attributes (repeat or comma-separate)

## Examples

```bash
# Rename a component
mass component update ecomm-db --name "Primary DB"

# Rename and retag it in one call
mass component update ecomm-db --name "Primary DB" -a priority=high

# Replace the full attribute set
mass component update ecomm-db -a priority=high,cost-center=engineering
```


```
mass component update <component-id> [flags]
```

### Options

```
  -a, --attributes stringToString   Replacement custom attributes (e.g. -a priority=high,cost-center=engineering) (default [])
  -d, --description string          New description
  -h, --help                        help for update
  -n, --name string                 New display name
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass component](/cli/commands/mass_component)	 - Manage components in a project's blueprint
