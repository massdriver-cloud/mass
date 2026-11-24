# Pause Event Destination

Stops sending events to a destination. Events recorded while it is paused wait, and are sent after `mass event-destination resume`. An event that waits for longer than 30 days (by default) becomes `FAILED`.

## Usage

```bash
mass event-destination pause <id>
```
