---
id: mass_resource_delete.md
slug: /cli/commands/mass_resource_delete
title: Mass Resource Delete
sidebar_label: Mass Resource Delete
---
## mass resource delete

Delete a resource

```
mass resource delete [resource-id] [flags]
```

### Examples

```
  # Delete an imported resource
  mass resource delete 12345678-1234-1234-1234-123456789012

  # Skip the confirmation prompt
  mass resource delete 12345678-1234-1234-1234-123456789012 --force
```

### Options

```
  -f, --force   Skip confirmation prompt
  -h, --help    help for delete
```

### SEE ALSO

* [mass resource](/cli/commands/mass_resource)	 - Manage resources
