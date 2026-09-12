---
id: mass_repository_grant_list.md
slug: /cli/commands/mass_repository_grant_list
title: Mass Repository Grant List
sidebar_label: Mass Repository Grant List
---
## mass repository grant list

List the sharing grants on a repository

### Synopsis

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


```
mass repository grant list <name> [flags]
```

### Options

```
  -h, --help            help for list
  -o, --output string   Output format (table, json) (default "table")
```

### SEE ALSO

* [mass repository grant](/cli/commands/mass_repository_grant)	 - Manage sharing grants on an OCI repository
