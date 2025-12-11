---
id: mass_definition_publish.md
slug: /cli/commands/mass_definition_publish
title: Mass Definition Publish
sidebar_label: Mass Definition Publish
---
## mass definition publish

Publish an artifact definition to Massdriver

### Synopsis

# Publish Artifact Definition

Publishes a new or updated artifact definition to Massdriver. Supports JSON or YAML formats.

## Usage

```bash
mass definition publish <definition-file>
```

## Examples

```bash
# Publish a definition from a JSON file
mass definition publish my-definition.json

# Publish a definition from a YAML file
mass definition publish my-definition.yaml
```


```
mass definition publish [flags]
```

### Options

```
  -h, --help   help for publish
```

### SEE ALSO

* [mass definition](/cli/commands/mass_definition)	 - Artifact definition management
