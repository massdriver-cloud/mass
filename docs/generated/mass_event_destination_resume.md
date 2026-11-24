---
id: mass_event_destination_resume.md
slug: /cli/commands/mass_event_destination_resume
title: Mass Event Destination Resume
sidebar_label: Mass Event Destination Resume
---
## mass event destination resume

Resume a paused event destination

### Synopsis

# Resume Event Destination

Starts sending events to a paused destination again, beginning with the events that waited. Use it after you fix a destination that Massdriver paused.

## Usage

```bash
mass event destination resume <id>
```


```
mass event destination resume <id> [flags]
```

### Options

```
  -h, --help   help for resume
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass event destination](/cli/commands/mass_event_destination)	 - Event destination management
