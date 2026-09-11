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

## Usage

```bash
mass resource-type publish [path]
```

`path` is a directory containing a `massdriver.yaml`, or the `massdriver.yaml`
itself (defaults to the current directory). Only `massdriver.yaml`, `readme`,
`changelog`, icon files, and the instruction/export template files referenced
by the `massdriver.yaml` are included in the published artifact.

## Examples

```bash
# Publish the resource type in the current directory
mass resource-type publish

# Publish a resource type from a specific directory
mass resource-type publish ./my-resource-type

# Or point directly at the massdriver.yaml
mass resource-type publish ./my-resource-type/massdriver.yaml
```

## Publishing a raw JSON schema (deprecated)

`path` may also point at a raw JSON (or YAML) schema file, the format that
predates `massdriver.yaml`:

```bash
mass resource-type publish ./my-resource-type.json
```

This is **deprecated** and will be removed in a future release. A raw schema has
no version of its own, so it is published as the resource type's unversioned
`0.0.0` document and cannot participate in resource type versioning.

Migrate with `mass resource-type convert`, which writes an equivalent
`massdriver.yaml` alongside the schema:

```bash
mass resource-type convert ./my-resource-type.json
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
