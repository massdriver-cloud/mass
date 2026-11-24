---
id: mass_event_destination_preview.md
slug: /cli/commands/mass_event_destination_preview
title: Mass Event Destination Preview
sidebar_label: Mass Event Destination Preview
---
## mass event destination preview

List recent events that event types, attributes, and a condition select

### Synopsis

# Preview Event Destination

Lists recent audit log events that event types, attributes, and a condition select, newest first. Use it to check them before you create or change a destination.

If the condition cannot run on an event, the preview stops with an error that names the event.

## Usage

```bash
mass event destination preview [flags]
```

## Flags

- `--event-type`: An event type, or `<entity>.*` for every event of an entity. Repeat for more.
- `--attribute`: An attribute value that the event's entity must have had, `key=value`. Repeat for more.
- `--condition`: A jq expression that must output `true`.
- `--limit`: The most events to list. Defaults to 25.
- `--output, -o`: Output format, `table` (default) or `json`

## Examples

```bash
mass event destination preview \
  --event-type 'deployment.*' \
  --attribute md-environment=prod \
  --condition '.data.metadata.deployment_action == "PROVISION"'
```


```
mass event destination preview [flags]
```

### Options

```
      --attribute stringArray    Attribute the event's entity must have had, key=value; repeat for more
      --condition string         jq expression that must output true
      --event-type stringArray   Event type, or <entity>.* for every event of an entity; repeat for more
  -h, --help                     help for preview
      --limit int                Most events to list (default 25)
  -o, --output string            Output format (table, json) (default "table")
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event destination](/cli/commands/mass_event_destination)	 - Event destination management
