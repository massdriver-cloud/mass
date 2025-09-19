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

Publishes a new or updated artifact definition to Massdriver.

## Usage

```bash
mass definition publish --file \<definition-file\>
```

## Examples

```bash
# Publish a definition from a file
mass definition publish --file my-definition.json

# Publish a definition from stdin
cat my-definition.json | mass definition publish --file -
```

## Options

- `--file`: Path to the definition file (use - for stdin)


```
mass definition publish [flags]
```

### Options

```
  -f, --file string   File containing artifact definition schema (use - for stdin)
  -h, --help          help for publish
```

### SEE ALSO

* [mass definition](/cli/commands/mass_definition)	 - Artifact definition management
