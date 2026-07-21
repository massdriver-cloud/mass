# Instance Remote References

A remote reference is a per-instance override of a single connection slot, pointing the slot at a resource from another project (or an imported resource) instead of the blueprint Link's wiring.

The override takes priority over any blueprint-level Link wired into the same slot, and reverts to the Link (or environment default) when removed.

## Subcommands

- `set` — override a connection slot with a specific resource
- `remove` — remove the override, reverting to the blueprint wiring

See `mass instance remote-reference set --help` and `mass instance remote-reference remove --help` for details.
