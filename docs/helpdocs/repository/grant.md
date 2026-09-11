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
