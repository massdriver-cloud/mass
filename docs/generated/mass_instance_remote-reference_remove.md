---
id: mass_instance_remote-reference_remove.md
slug: /cli/commands/mass_instance_remote-reference_remove
title: Mass Instance Remote-Reference Remove
sidebar_label: Mass Instance Remote-Reference Remove
---
## mass instance remote-reference remove

Remove a connection slot override, reverting to the blueprint wiring

### Synopsis

# Remove an Instance Remote Reference

Removes a per-instance connection slot override, reverting the slot to the blueprint Link's wiring (or the environment default).

Like other configuration changes, the instance must not be in PROVISIONED or FAILED status.

## Usage

```bash
mass instance remote-reference remove <instance-id> <field>
```

Where `<field>` is the connection slot key whose override should be removed.

## Examples

```bash
mass instance remote-reference remove ecomm-prod-api database
```


```
mass instance remote-reference remove <instance-id> <field> [flags]
```

### Examples

```
mass instance remote-reference remove ecomm-prod-api database
```

### Options

```
  -h, --help   help for remove
```

### SEE ALSO

* [mass instance remote-reference](/cli/commands/mass_instance_remote-reference)	 - Manage an instance's remote-reference connection overrides
