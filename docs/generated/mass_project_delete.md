---
id: mass_project_delete.md
slug: /cli/commands/mass_project_delete
title: Mass Project Delete
sidebar_label: Mass Project Delete
---
## mass project delete

Delete a project

### Synopsis

# Delete Project

Permanently deletes a project and all its resources. This action cannot be undone.

## Usage

```bash
mass project delete <project> [flags]
```

Where `<project>` is the project ID or slug.

## Flags

- `--force, -f`: Skip confirmation prompt (useful for automation)

## Confirmation

By default, this command requires typing "yes" to confirm deletion. This is a safety measure to prevent accidental deletions. Use the `--force` flag to skip this confirmation.

## Examples

```bash
# Delete a project by slug (with confirmation)
mass project delete myproject

# Delete a project by ID (with confirmation)
mass project delete 12345678-1234-1234-1234-123456789012

# Delete a project without confirmation prompt
mass project delete myproject --force
```


```
mass project delete [project] [flags]
```

### Options

```
  -f, --force   Skip confirmation prompt
  -h, --help    help for delete
```

### SEE ALSO

* [mass project](/cli/commands/mass_project)	 - Project management
