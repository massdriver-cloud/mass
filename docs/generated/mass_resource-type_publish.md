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

Publishes a new or updated resource type to Massdriver. Supports JSON or YAML formats.

## Usage

```bash
mass resource-type publish <resource-type-file>
```

## Examples

```bash
# Publish a resource type from a JSON file
mass resource-type publish my-resource-type.json

# Publish a resource type from a YAML file
mass resource-type publish my-resource-type.yaml
```


```
mass resource-type publish [resource-type file] [flags]
```

### Options

```
  -h, --help   help for publish
```

### SEE ALSO

* [mass resource-type](/cli/commands/mass_resource-type)	 - Resource type management
