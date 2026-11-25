# Set Stash Entry

Stores a value under a key. A value of `-` reads the value from stdin and keeps every byte, including a trailing newline.

Each write replaces the entry's value, format, TTL, and flags.

## Usage

```bash
mass stash set <scope>.<key> <value> [flags]
```

## Flags

- `--format`: `text` (default) or `json`. A `json` value must be a JSON object.
- `--ttl`: Delete the entry this many seconds after the write
- `--immutable`: Keep the first value. Later writes fail.
- `--sensitive`: Hide the value from callers that cannot write the scope

## Examples

```bash
mass stash set "$DEPLOYMENT_ID.db/hostname" "$DB_HOST"

# Run migrations once per deployment
mass stash set "$DEPLOYMENT_ID.migrations/ran" "true" --immutable

mass stash set "$DEPLOYMENT_ID.stats" "$(jq -n '{duration_s: 412, changed: 3}')" --format json

# Keep for one day
mass stash set "api-staging-db.ami-id" "ami-0abc" --ttl 86400

terraform output -raw kubeconfig | mass stash set "api-prod.kubeconfig" - --sensitive
```
