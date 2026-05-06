# Delete Resource Type

Permanently deletes a resource type from Massdriver. This action cannot be undone.

**Warning:** Resource types cannot be deleted if they are in use by any bundles or provisioned resources. This command requires administrator permissions.

## Usage

```bash
mass resource-type delete <resource-type> [flags]
```

Where `<resource-type>` is the resource type name (e.g., `aws-s3` or `my-org/aws-s3`).

## Flags

- `--force, -f`: Skip confirmation prompt (useful for automation)

## Confirmation

By default, this command requires typing the resource type name to confirm deletion. This is a safety measure to prevent accidental deletions. Use the `--force` flag to skip this confirmation.

## Examples

```bash
# Delete a resource type by name (with confirmation)
mass resource-type delete aws-s3

# Delete a resource type by full name (with confirmation)
mass resource-type delete my-org/aws-s3

# Delete a resource type without confirmation prompt
mass resource-type delete aws-s3 --force
```

## Notes

- This command will fail if the resource type is in use by any bundles or provisioned resources
- Administrator permissions are required to delete resource types
- The resource type name can be specified as a short name (e.g., `aws-s3`) or a full name (e.g., `my-org/aws-s3`)
