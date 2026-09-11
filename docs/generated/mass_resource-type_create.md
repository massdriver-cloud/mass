---
id: mass_resource-type_create.md
slug: /cli/commands/mass_resource-type_create
title: Mass Resource-Type Create
sidebar_label: Mass Resource-Type Create
---
## mass resource-type create

Create a new resource type OCI repository in your organization's catalog

### Synopsis

# Create Resource Type

Creates a new resource type OCI repository in your organization's catalog. The
repository starts empty; publish a version to it with
`mass resource-type publish`.

## Usage

```bash
mass resource-type create <name>
```

## Examples

```bash
# Create a resource type repository
mass resource-type create my-resource-type

# Create with custom attributes
mass resource-type create my-resource-type -a owner=data,service=database
```


```
mass resource-type create <name> [flags]
```

### Options

```
  -a, --attributes stringToString   Custom attributes (e.g. -a owner=data,service=database) (default [])
  -h, --help                        help for create
```

### SEE ALSO

* [mass resource-type](/cli/commands/mass_resource-type)	 - Resource type management
