---
id: mass_resource_grant_delete.md
slug: /cli/commands/mass_resource_grant_delete
title: Mass Resource Grant Delete
sidebar_label: Mass Resource Grant Delete
---
## mass resource grant delete

Revoke a sharing grant by id

### Synopsis

# Delete Resource Grant

Revokes a sharing grant by id. Environments that qualified only through this grant immediately lose access to the resource.

## Usage

```bash
mass resource grant delete <grant-id>
```

## Examples

```bash
# Find the grant id, then revoke it
mass resource grant list api-prod-database-connection
mass resource grant delete 4f2a9c18-1f0e-4a5c-9a3e-2b6d7c8e9f10
```

## Notes

- Grants are immutable, so changing one means deleting it and creating a replacement.
- This does not prompt for confirmation. A revoked grant can be re-created with `mass resource grant create`.


```
mass resource grant delete <grant-id> [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass resource grant](/cli/commands/mass_resource_grant)	 - Manage sharing grants on a resource
