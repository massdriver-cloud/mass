---
id: mass_resource-type_publish.md
slug: /cli/commands/mass_resource-type_publish
title: Mass Resource-Type Publish
sidebar_label: Mass Resource-Type Publish
---
## mass resource-type publish

Publish a resource type to Massdriver

### Synopsis

# Publish Resource Type

Publishes a resource type to your organization's catalog as an OCI artifact.

The resource type is authored as a `massdriver.yaml` file, which must include a
`version` field. Publishing is immutable: a version that already exists cannot be
republished.

Raw JSON schema publishing is no longer supported. If you have a raw JSON schema,
convert it first with `mass resource-type convert`.

## Usage

```bash
mass resource-type publish [path]
```

`path` is a directory containing a `massdriver.yaml` (defaults to the current
directory). Only `massdriver.yaml`, `readme`, `changelog`, icon files, and the
instruction/export template files referenced by the `massdriver.yaml` are
included in the published artifact.

## Examples

```bash
# Publish the resource type in the current directory
mass resource-type publish

# Publish a resource type from a specific directory
mass resource-type publish ./my-resource-type
```


```
mass resource-type publish [path] [flags]
```

### Options

```
  -h, --help   help for publish
```

### SEE ALSO

* [mass resource-type](/cli/commands/mass_resource-type)	 - Resource type management
