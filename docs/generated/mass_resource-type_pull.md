---
id: mass_resource-type_pull.md
slug: /cli/commands/mass_resource-type_pull
title: Mass Resource-Type Pull
sidebar_label: Mass Resource-Type Pull
---
## mass resource-type pull

Pull a resource type from Massdriver to a local directory

### Synopsis

# Pull Resource Type

Pulls a published resource type from your organization's catalog into a local
directory.

## Usage

```bash
mass resource-type pull <resource-type> [flags]
```

## Examples

```bash
# Pull the latest version into a directory named after the resource type
mass resource-type pull my-resource-type

# Pull a specific version into a specific directory
mass resource-type pull my-resource-type --version 1.2.0 --directory ./out
```


```
mass resource-type pull <resource-type> [flags]
```

### Options

```
  -d, --directory string   Directory to output the resource type. Defaults to the resource type name.
  -f, --force              Force pull even if the directory already exists. This will overwrite existing files.
  -h, --help               help for pull
  -v, --version string     Resource type version or release channel (default "latest")
```

### SEE ALSO

* [mass resource-type](/cli/commands/mass_resource-type)	 - Resource type management
