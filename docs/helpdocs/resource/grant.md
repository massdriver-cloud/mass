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
