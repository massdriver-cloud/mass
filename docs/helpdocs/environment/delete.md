# Delete Environment

Permanently deletes an environment. This action cannot be undone.

Deleting an environment does not tear down its running infrastructure. Decommission the environment's instances first with `mass environment decommission` if you want the underlying resources removed.

## Usage

```bash
mass environment delete <environment> [flags]
```

Where `<environment>` is the environment ID.

## Flags

- `--force, -f`: Skip confirmation prompt (useful for automation)

## Confirmation

By default, this command requires typing the environment ID to confirm deletion. This is a safety measure to prevent accidental deletions. Use the `--force` flag to skip this confirmation.

## Examples

```bash
# Delete an environment (with confirmation)
mass environment delete ecomm-staging

# Delete an environment without confirmation prompt
mass environment delete ecomm-staging --force
```
