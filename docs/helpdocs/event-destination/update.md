# Update Event Destination

Changes an event destination. Only the flags you set change.

## Usage

```bash
mass event-destination update <id> [flags]
```

## Flags

- `--name`: New name of the destination.
- `--url`: New URL. Changes `url` in the current config.
- `--header`: A header sent with each event, `Name=value`. Adds to or changes `headers` in the current config. Repeat for more. To remove a header, use `--config`.
- `--config`: Replaces the config, as JSON or `@file`.
- `--auth`: Replaces the credentials, as JSON or `@file`. `--auth '{}'` removes them.
- `--event-type`: Replaces the event types. Repeat for more. `--event-type ''` receives every event.
- `--attribute`: Replaces the attributes, `key=value`. Repeat for more. `--attribute ''` receives events of any attributes.
- `--condition`: Replaces the jq condition. `--condition ''` removes it.

New event types and attributes apply to events recorded after the change. A new condition also applies to events that wait to be sent.

## Examples

```bash
# Rotate the credentials
mass event-destination update 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 --auth @auth.json

# Receive every event
mass event-destination update 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 --event-type ''
```
