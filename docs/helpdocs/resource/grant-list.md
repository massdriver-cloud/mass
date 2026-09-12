# List Resource Grants

Lists the sharing grants authored on a resource — what it is shared as, and which recipient environments qualify.

If you can see a resource you can see all of its grants; they are publisher-side metadata rather than being visibility-gated themselves.

## Usage

```bash
mass resource grant list <resource-id> [flags]
```

## Flags

- `-o, --output` — output format: `table` or `json`

## Examples

```bash
# List the grants on a resource
mass resource grant list api-prod-database-connection

# As JSON, for scripting
mass resource grant list api-prod-database-connection -o json
```

## Notes

- The `Recipients` column reads `everyone` for an organization-wide grant, `key=*` for a per-key wildcard, and `key=a|b` for a closed set.
- The `ID` column is what `mass resource grant delete` takes.
