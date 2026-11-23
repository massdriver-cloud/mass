# List Event Types

Lists the event types, sorted by id.

## Usage

```bash
mass event-type list [flags]
```

## Flags

- `--entity`: Only event types of this entity, e.g. `deployment`
- `--output, -o`: Output format, `table` (default) or `json`. `json` includes the metadata each event type records.

## Examples

```bash
mass event-type list --entity deployment
```
