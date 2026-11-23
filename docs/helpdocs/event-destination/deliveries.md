# List Event Destination Deliveries

Lists a destination's deliveries, newest first. A destination has one delivery for each event it selects.

- `DELIVERED`: the destination accepted the event.
- `SKIPPED`: the condition did not pass the event.
- `FAILED`: no more attempts. `mass event-destination redeliver` sends it again.
- `PENDING`: waits to be sent or retried.

## Usage

```bash
mass event-destination deliveries <id> [flags]
```

## Flags

- `--status`: Only deliveries with this status. Repeat for more.
- `--output, -o`: Output format, `table` (default) or `json`

## Examples

```bash
mass event-destination deliveries 4f9b2c1e-8d3a-4c6e-9f1b-2a7d5e8c0b13 --status FAILED
```
