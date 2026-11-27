# List Stash Entries

Lists the entries in one scope. Sensitive values show as `(sensitive)` unless you can write the scope.

## Usage

```bash
mass alpha stash list <scope> [flags]
```

## Flags

- `--output, -o`: Output format, `table` (default) or `json`

## Examples

```bash
mass alpha stash list api-prod

mass alpha stash list "$DEPLOYMENT_ID" -o json | jq -r '.[].key'
```
