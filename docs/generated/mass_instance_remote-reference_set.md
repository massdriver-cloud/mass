---
id: mass_instance_remote-reference_set.md
slug: /cli/commands/mass_instance_remote-reference_set
title: Mass Instance Remote-Reference Set
sidebar_label: Mass Instance Remote-Reference Set
---
## mass instance remote-reference set

Override a connection slot with a resource from another project

### Synopsis

# Set an Instance Remote Reference

Overrides one of an instance's connection slots with a resource from another project (or an imported resource).

The override takes priority over any blueprint-level Link wired into the same slot, and reverts to the Link (or environment default) when removed with `mass instance remote-reference remove`.

Like other configuration changes, the instance must not be in PROVISIONED or FAILED status.

## Usage

```bash
mass instance remote-reference set <instance-id> <field> <resource-id>
```

- `<field>` is the key in the instance's bundle `connectionsSchema` to bind.
- `<resource-id>` is either a UUID (for imported resources) or `instance.field` (for provisioned resources).

## Examples

```bash
# Point the "database" connection slot at a resource produced by another instance
mass instance remote-reference set ecomm-prod-api database ecomm-prod-db.postgres

# Point a slot at an imported resource by UUID
mass instance remote-reference set ecomm-prod-api database 12345678-1234-1234-1234-123456789012
```


```
mass instance remote-reference set <instance-id> <field> <resource-id> [flags]
```

### Examples

```
mass instance remote-reference set ecomm-prod-api database ecomm-prod-db.postgres
```

### Options

```
  -h, --help   help for set
```

### SEE ALSO

* [mass instance remote-reference](/cli/commands/mass_instance_remote-reference)	 - Manage an instance's remote-reference connection overrides
