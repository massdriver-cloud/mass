# Get Stash Entry

Prints an entry's value exactly as it is stored, with no trailing newline.

A sensitive value is shown only to callers that can write its scope. Other callers get an error.

## Usage

```bash
mass alpha stash get <scope>.<key> [flags]
```

## Flags

- `--output, -o`: Output format, `text` (default) or `json`. `json` prints the whole entry: its value, label, format, flags, and expiry.

## Examples

```bash
DB_HOST=$(mass alpha stash get "$DEPLOYMENT_ID.db/hostname")

mass alpha stash get "$DEPLOYMENT_ID.stats" -o json | jq .data.changed
```
