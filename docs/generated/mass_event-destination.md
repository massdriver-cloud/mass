---
id: mass_event-destination.md
slug: /cli/commands/mass_event-destination
title: Mass Event-Destination
sidebar_label: Mass Event-Destination
---
## mass event-destination

Event destination management

### Synopsis

# Manage Event Destinations

[Event destinations](https://docs.massdriver.cloud/platform-operations/event-destinations) send your organization's audit log events to a place outside Massdriver, such as an HTTP endpoint.

A destination selects events by event type and by the attributes of the event's entity. A jq condition can drop more of them. `mass event-type list` lists the event types.

Massdriver pauses a destination when its sends fail for 24 hours (by default). Events wait while a destination is paused, and are sent when it is resumed.


### Options

```
  -h, --help   help for event-destination
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass event-destination create](/cli/commands/mass_event-destination_create)	 - Create an event destination
* [mass event-destination delete](/cli/commands/mass_event-destination_delete)	 - Delete an event destination
* [mass event-destination deliveries](/cli/commands/mass_event-destination_deliveries)	 - List an event destination's deliveries
* [mass event-destination get](/cli/commands/mass_event-destination_get)	 - Get an event destination
* [mass event-destination list](/cli/commands/mass_event-destination_list)	 - List event destinations
* [mass event-destination pause](/cli/commands/mass_event-destination_pause)	 - Pause an event destination
* [mass event-destination preview](/cli/commands/mass_event-destination_preview)	 - List recent events that event types, attributes, and a condition select
* [mass event-destination redeliver](/cli/commands/mass_event-destination_redeliver)	 - Send failed deliveries again
* [mass event-destination resume](/cli/commands/mass_event-destination_resume)	 - Resume a paused event destination
* [mass event-destination update](/cli/commands/mass_event-destination_update)	 - Update an event destination
