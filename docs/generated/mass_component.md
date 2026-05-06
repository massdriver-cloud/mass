---
id: mass_component.md
slug: /cli/commands/mass_component
title: Mass Component
sidebar_label: Mass Component
---
## mass component

Manage components in a project's blueprint

### Synopsis

# Components

A project contains a **blueprint** — the design-time architecture of your infrastructure. Components are the slots in that blueprint, each backed by a bundle. **Links** describe how one component's output wires into another component's input.

At deploy time, each environment materializes the blueprint into live **instances** and **connections**.

Use these commands to manage components and links in a project's blueprint:

- `mass component add` — add a component to a project
- `mass component remove` — remove a component
- `mass component link` — link two components together
- `mass component unlink` — remove a link


### Options

```
  -h, --help   help for component
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass component add](/cli/commands/mass_component_add)	 - Add a component to a project's blueprint
* [mass component link](/cli/commands/mass_component_link)	 - Link two components in a project's blueprint
* [mass component remove](/cli/commands/mass_component_remove)	 - Remove a component from a project's blueprint
* [mass component unlink](/cli/commands/mass_component_unlink)	 - Remove a link between two components
* [mass component update](/cli/commands/mass_component_update)	 - Update a component's name, description, or attributes
