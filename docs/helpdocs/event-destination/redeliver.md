# Redeliver Events

Sends a destination's `FAILED` deliveries again, with a new set of retries. Deliveries that are not `FAILED` do not change. A paused destination sends them after it is resumed.

## Usage

```bash
mass event-destination redeliver <id> [flags]
```

## Flags

- `--delivery`: A failed delivery id to send again. Repeat for more. Leave it out to send every failed delivery again.

## Examples

```bash
# Send every failed delivery again
mass event-destination redeliver 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13

# Send one delivery again
mass event-destination redeliver 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 --delivery 7c1d0e2f-5a4b-4f3e-8d2c-1b0a9f8e7d6c
```
