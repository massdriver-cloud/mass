# Delete Stash Entry

Deletes an entry. You must type the key to confirm, unless you set `--force`. A different answer cancels the delete and exits 0, so scripts should use `--force`.

## Usage

```bash
mass alpha stash delete <scope>.<key> [flags]
```

## Flags

- `--force, -f`: Skip the confirmation prompt

## Examples

```bash
mass alpha stash delete "api-staging-db.ami-id"

mass alpha stash delete "env:api-db.config/v2" --force
```
