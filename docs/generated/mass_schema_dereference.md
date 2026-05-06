---
id: mass_schema_dereference.md
slug: /cli/commands/mass_schema_dereference
title: Mass Schema Dereference
sidebar_label: Mass Schema Dereference
---
## mass schema dereference

Dereferences a JSON Schema

### Synopsis

# Dereferences a JSON Schema Document

This command will expand all the `$ref` statements in a JSON Schema. This command is useful when managing resource type schemas and using `$refs` to keep your schemas "DRY".

## Examples

From an existing file

```shell
mass schema dereference --file resource-type.json
```

From stdin

```shell
cat resource-type.json | mass schema dereference -f -
```


```
mass schema dereference [flags]
```

### Options

```
  -f, --file string   Path to JSON document
  -h, --help          help for dereference
```

### SEE ALSO

* [mass schema](/cli/commands/mass_schema)	 - Manage JSON Schemas
