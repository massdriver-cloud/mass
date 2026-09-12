# Delete Resource

Deletes an imported resource from your organization.

Only imported resources can be deleted this way. Provisioned resources belong to the deployment that created them — remove those with `mass instance destroy`.

## Usage

```bash
mass resource delete <resource-id> [flags]
```

## Flags

- `-f, --force` — skip the confirmation prompt

## Examples

```bash
# Delete an imported resource, confirming at the prompt
mass resource delete 12345678-1234-1234-1234-123456789012

# Skip the confirmation prompt
mass resource delete 12345678-1234-1234-1234-123456789012 --force
```
