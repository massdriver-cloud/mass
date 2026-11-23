---
id: mass_event-destination_deliveries.md
slug: /cli/commands/mass_event-destination_deliveries
title: Mass Event-Destination Deliveries
sidebar_label: Mass Event-Destination Deliveries
---
## mass event-destination deliveries

List an event destination's deliveries

### Synopsis

# List Event Destination Deliveries

Lists a destination's deliveries, newest first. A destination has one delivery for each event it selects.

- `DELIVERED`: the destination accepted the event.
- `SKIPPED`: the condition did not pass the event.
- `FAILED`: no more attempts. `mass event-destination redeliver` sends it again.
- `PENDING`: waits to be sent or retried.

## Usage

```bash
mass event-destination deliveries <id> [flags]
```

## Flags

- `--status`: Only deliveries with this status. Repeat for more.
- `--output, -o`: Output format, `table` (default) or `json`

## Examples

```bash
mass event-destination deliveries 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 --status FAILED
```


```
mass event-destination deliveries <id> [flags]
```

### Options

```
  -h, --help                 help for deliveries
  -o, --output string        Output format (table, json) (default "table")
      --status stringArray   Only deliveries with this status (DELIVERED, SKIPPED, FAILED, or PENDING); repeat for more
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event-destination](/cli/commands/mass_event-destination)	 - Event destination management
