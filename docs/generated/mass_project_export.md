---
id: mass_project_export.md
slug: /cli/commands/mass_project_export
title: Mass Project Export
sidebar_label: Mass Project Export
---
## mass project export

Export a project from Massdriver

### Synopsis

# Export Project Details

Exports a project to the local filesystem. This is useful for backups and migrations.

Data will be exported in a tiered file structure, with directory names as the project/environment/instance slugs:

```bash
project
├── env1
│   ├── instance1
│   │   ├── <data...>
│   ├── instance2
│   │   ├── <data...>
├── env2
│   ├── instance1
│   │   ├── <data...>
│   ├── instance2
│   │   ├── <data...>
```

For information about what will be exported for each environment, refer to the `mass environment export` command.
For information about what will be exported for each instance, refer to the `mass instance export` command.

## Usage

```bash
mass project export <project-slug>
```

## Examples

```bash
# Export the "web" project
mass project export web
```


```
mass project export [project] [flags]
```

### Options

```
  -h, --help   help for export
```

### SEE ALSO

* [mass project](/cli/commands/mass_project)	 - Project management
