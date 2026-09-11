---
id: mass_repository_grant.md
slug: /cli/commands/mass_repository_grant
title: Mass Repository Grant
sidebar_label: Mass Repository Grant
---
## mass repository grant

Manage sharing grants on an OCI repository

### Synopsis

# Repository Grants

Manages the sharing grants on an OCI repository.

A grant says "this repository is shared as `<action>`, to recipient projects matching `<conditions>`." Without a grant, a repository is visible only where it was published.

Grants are immutable. To change an action or its conditions, delete the grant and create a replacement.

## Usage

```bash
mass repository grant create <name> [flags]
mass repository grant list <name> [flags]
mass repository grant delete <grant-id>
```

## Notes

- Creating or deleting a grant requires the `repo:grant` action on the repository.
- Recipients are matched on **project** attributes. Resource grants match on environment attributes instead — see `mass resource grant`.


### Options

```
  -h, --help   help for grant
```

### SEE ALSO

* [mass repository](/cli/commands/mass_repository)	 - Manage OCI repositories (bundles and resource types)
* [mass repository grant create](/cli/commands/mass_repository_grant_create)	 - Share a repository with recipient projects
* [mass repository grant delete](/cli/commands/mass_repository_grant_delete)	 - Revoke a sharing grant by id
* [mass repository grant list](/cli/commands/mass_repository_grant_list)	 - List the sharing grants on a repository
