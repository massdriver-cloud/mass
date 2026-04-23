# Delete Artifact Definition

Permanently deletes an artifact definition from Massdriver. This action cannot be undone.

**Warning:** Artifact definitions cannot be deleted if they are in use by any bundles or provisioned artifacts. This command requires administrator permissions.

## Usage

```bash
mass definition delete <definition-name> [flags]
```

Where `<definition-name>` is the artifact definition name (e.g., `aws-s3` or `my-org/aws-s3`).

## Flags

- `--force, -f`: Skip confirmation prompt (useful for automation)

## Confirmation

By default, this command requires typing the artifact definition name to confirm deletion. This is a safety measure to prevent accidental deletions. Use the `--force` flag to skip this confirmation.

## Examples

```bash
# Delete an artifact definition by name (with confirmation)
mass definition delete aws-s3

# Delete an artifact definition by full name (with confirmation)
mass definition delete my-org/aws-s3

# Delete an artifact definition without confirmation prompt
mass definition delete aws-s3 --force
```

## Notes

- This command will fail if the artifact definition is in use by any bundles or provisioned artifacts
- Administrator permissions are required to delete artifact definitions
- The definition name can be specified as a short name (e.g., `aws-s3`) or a full name (e.g., `my-org/aws-s3`)
