---
id: mass_event.md
slug: /cli/commands/mass_event
title: Mass Event
sidebar_label: Mass Event
---
## mass event

Events, event types, and event destinations

### Synopsis

# Events

Massdriver records each change in your organization as an event in the audit log. `mass event type list` lists the event types, and `mass event destination` sends events to a place outside Massdriver.


### Options

```
  -h, --help   help for event
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass event destination](/cli/commands/mass_event_destination)	 - Event destination management
* [mass event type](/cli/commands/mass_event_type)	 - Event catalog
