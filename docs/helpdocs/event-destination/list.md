# List Event Destinations

Lists the event destinations in your organization.

## Usage

```bash
mass event-destination list [flags]
```

## Flags

- `--status`: Only destinations with this status, `ACTIVE` or `PAUSED`. Repeat for more.
- `--output, -o`: Output format, `table` (default) or `json`

## Examples

```bash
# List all destinations
mass event-destination list

# List the paused destinations
mass event-destination list --status PAUSED
```
