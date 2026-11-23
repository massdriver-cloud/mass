---
id: mass_event-type_list.md
slug: /cli/commands/mass_event-type_list
title: Mass Event-Type List
sidebar_label: Mass Event-Type List
---
## mass event-type list

List event types

### Synopsis

# List Event Types

Lists the event types, sorted by id.

## Usage

```bash
mass event-type list [flags]
```

## Flags

- `--entity`: Only event types of this entity, e.g. `deployment`
- `--output, -o`: Output format, `table` (default) or `json`. `json` includes the metadata each event type records.

## Examples

```bash
mass event-type list --entity deployment
```


```
mass event-type list [flags]
```

### Options

```
      --entity string   Only event types of this entity, e.g. deployment
  -h, --help            help for list
  -o, --output string   Output format (table, json) (default "table")
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event-type](/cli/commands/mass_event-type)	 - Event catalog
