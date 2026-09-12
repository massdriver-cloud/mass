---
id: mass_resource_grant_list.md
slug: /cli/commands/mass_resource_grant_list
title: Mass Resource Grant List
sidebar_label: Mass Resource Grant List
---
## mass resource grant list

List the sharing grants on a resource

### Synopsis

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


```
mass resource grant list <resource-id> [flags]
```

### Options

```
  -h, --help            help for list
  -o, --output string   Output format (table, json) (default "table")
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass resource grant](/cli/commands/mass_resource_grant)	 - Manage sharing grants on a resource
