# Delete Event Destination

Deletes an event destination and its deliveries. Events that wait to be sent are not sent.

You must type the destination's name to confirm, unless you set `--force`.

## Usage

```bash
mass event-destination delete <id> [flags]
```

## Flags

- `--force, -f`: Skip the confirmation prompt
