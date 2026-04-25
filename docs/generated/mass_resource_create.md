---
id: mass_resource_create.md
slug: /cli/commands/mass_resource_create
title: Mass Resource Create
sidebar_label: Mass Resource Create
---
## mass resource create

Create a resource

### Synopsis

# Create a resource

Create a resource to represent infrastructure not deployed through Massdriver.

## Examples

```shell
mass resource create -n <name> -t <type> -f <file>
```


```
mass resource create [flags]
```

### Options

```
  -f, --file string   Resource file
  -h, --help          help for create
  -n, --name string   Resource name
  -t, --type string   Resource type
```

### SEE ALSO

* [mass resource](/cli/commands/mass_resource)	 - Manage resources
