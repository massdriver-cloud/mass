# Manage Event Destinations

[Event destinations](https://docs.massdriver.cloud/platform-operations/event-destinations) send your organization's audit log events to a place outside Massdriver, such as an HTTP endpoint.

A destination selects events by event type and by the attributes of the event's entity. A jq condition can drop more of them. `mass event-type list` lists the event types.

Massdriver pauses a destination when its sends fail for 24 hours. Events wait while a destination is paused, and are sent when it is resumed.
