---
id: mass_event_destination_delete.md
slug: /cli/commands/mass_event_destination_delete
title: Mass Event Destination Delete
sidebar_label: Mass Event Destination Delete
---
## mass event destination delete

Delete an event destination

### Synopsis

# Delete Event Destination

Deletes an event destination and its deliveries. Events that wait to be sent are not sent.

You must type the destination's name to confirm, unless you set `--force`.

## Usage

```bash
mass event destination delete <id> [flags]
```

## Flags

- `--force, -f`: Skip the confirmation prompt


```
mass event destination delete <id> [flags]
```

### Options

```
  -f, --force   Skip confirmation prompt
  -h, --help    help for delete
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event destination](/cli/commands/mass_event_destination)	 - Event destination management
