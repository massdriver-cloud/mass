---
id: mass_event_destination_list.md
slug: /cli/commands/mass_event_destination_list
title: Mass Event Destination List
sidebar_label: Mass Event Destination List
---
## mass event destination list

List event destinations

### Synopsis

# List Event Destinations

Lists the event destinations in your organization.

## Usage

```bash
mass event destination list [flags]
```

## Flags

- `--status`: Only destinations with this status, `ACTIVE` or `PAUSED`. Repeat for more.
- `--output, -o`: Output format, `table` (default) or `json`

## Examples

```bash
# List all destinations
mass event destination list

# List the paused destinations
mass event destination list --status PAUSED
```


```
mass event destination list [flags]
```

### Options

```
  -h, --help                 help for list
  -o, --output string        Output format (table, json) (default "table")
      --status stringArray   Only destinations with this status (ACTIVE or PAUSED); repeat for more
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event destination](/cli/commands/mass_event_destination)	 - Event destination management
