---
id: mass_instance_remote-reference.md
slug: /cli/commands/mass_instance_remote-reference
title: Mass Instance Remote-Reference
sidebar_label: Mass Instance Remote-Reference
---
## mass instance remote-reference

Manage an instance's remote-reference connection overrides

### Synopsis

# Instance Remote References

A remote reference is a per-instance override of a single connection slot, pointing the slot at a resource from another project (or an imported resource) instead of the blueprint Link's wiring.

The override takes priority over any blueprint-level Link wired into the same slot, and reverts to the Link (or environment default) when removed.

## Subcommands

- `set` — override a connection slot with a specific resource
- `remove` — remove the override, reverting to the blueprint wiring

See `mass instance remote-reference set --help` and `mass instance remote-reference remove --help` for details.


### Options

```
  -h, --help   help for remote-reference
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
* [mass instance remote-reference remove](/cli/commands/mass_instance_remote-reference_remove)	 - Remove a connection slot override, reverting to the blueprint wiring
* [mass instance remote-reference set](/cli/commands/mass_instance_remote-reference_set)	 - Override a connection slot with a resource from another project
