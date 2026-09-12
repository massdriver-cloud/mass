---
id: mass_resource_grant.md
slug: /cli/commands/mass_resource_grant
title: Mass Resource Grant
sidebar_label: Mass Resource Grant
---
## mass resource grant

Manage sharing grants on a resource

### Synopsis

# Resource Grants

Manages the sharing grants on a resource.

A grant says "this resource is shared as `<action>`, to recipient environments matching `<conditions>`." Without a grant, a resource is visible only where it was created.

Grants are immutable. To change an action or its conditions, delete the grant and create a replacement.

## Usage

```bash
mass resource grant create <resource-id> [flags]
mass resource grant list <resource-id> [flags]
mass resource grant delete <grant-id>
```

## Notes

- Creating or deleting a grant requires the `resource:grant` action on the resource.
- Recipients are matched on **environment** attributes. Repository grants match on project attributes instead — see `mass repository grant`.


### Options

```
  -h, --help   help for grant
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass resource](/cli/commands/mass_resource)	 - Manage resources
* [mass resource grant create](/cli/commands/mass_resource_grant_create)	 - Share a resource with recipient environments
* [mass resource grant delete](/cli/commands/mass_resource_grant_delete)	 - Revoke a sharing grant by id
* [mass resource grant list](/cli/commands/mass_resource_grant_list)	 - List the sharing grants on a resource
