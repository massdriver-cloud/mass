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
