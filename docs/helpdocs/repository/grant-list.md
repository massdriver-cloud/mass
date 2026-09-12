# List Repository Grants

Lists the sharing grants authored on an OCI repository — what it is shared as, and which recipient projects qualify.

If you can see a repository you can see all of its grants; they are publisher-side metadata rather than being visibility-gated themselves.

## Usage

```bash
mass repository grant list <name> [flags]
```

## Flags

- `-o, --output` — output format: `table` or `json`

## Examples

```bash
# List the grants on a repository
mass repository grant list aws-aurora-postgres

# As JSON, for scripting
mass repository grant list aws-aurora-postgres -o json
```

## Notes

- The `Recipients` column reads `everyone` for an organization-wide grant, `key=*` for a per-key wildcard, and `key=a|b` for a closed set.
- The `ID` column is what `mass repository grant delete` takes.
