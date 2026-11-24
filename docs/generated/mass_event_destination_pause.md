---
id: mass_event_destination_pause.md
slug: /cli/commands/mass_event_destination_pause
title: Mass Event Destination Pause
sidebar_label: Mass Event Destination Pause
---
## mass event destination pause

Pause an event destination

### Synopsis

# Pause Event Destination

Stops sending events to a destination. Events recorded while it is paused wait, and are sent after `mass event destination resume`. An event that waits for longer than 30 days (by default) becomes `FAILED`.

## Usage

```bash
mass event destination pause <id>
```


```
mass event destination pause <id> [flags]
```

### Options

```
  -h, --help   help for pause
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event destination](/cli/commands/mass_event_destination)	 - Event destination management
