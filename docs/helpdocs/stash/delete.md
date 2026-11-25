# Delete Stash Entry

Deletes an entry. You must type the key to confirm, unless you set `--force`.

## Usage

```bash
mass stash delete <scope>.<key> [flags]
```

## Flags

- `--force, -f`: Skip the confirmation prompt

## Examples

```bash
mass stash delete "api-staging-db.ami-id"

mass stash delete "env:api-db.config/v2" --force
```
