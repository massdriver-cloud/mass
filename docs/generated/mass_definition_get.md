---
id: mass_definition_get.md
slug: /cli/commands/mass_definition_get
title: Mass Definition Get
sidebar_label: Mass Definition Get
---
## mass definition get

Get an artifact definition from Massdriver

### Synopsis

# Get Artifact Definition

Retrieves detailed information about a specific artifact definition, including:
- Definition name and label
- Schema
- UI configuration
- Connection settings

## Usage

```bash
mass definition get \<definition-name\>
```

## Examples

```bash
# Get details for the "aws-s3" definition
mass definition get aws-s3
```

## Options

- `--output`: Output format (text or json)


```
mass definition get [definition] [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass definition](/cli/commands/mass_definition)	 - Artifact definition management
