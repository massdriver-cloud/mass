---
id: mass_project_get.md
slug: /cli/commands/mass_project_get
title: Mass Project Get
sidebar_label: Mass Project Get
---
## mass project get

Get a project from Massdriver

### Synopsis

# Get Project Details

Retrieves detailed information about a specific project, including:
- Project name and description
- Environment configurations
- Cost metrics
- Project settings

## Usage

```bash
mass project get \<project-slug\>
```

## Examples

```bash
# Get details for the "alarms" project
mass project get alarms
```


```
mass project get [project] [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass project](/cli/commands/mass_project)	 - Project management
