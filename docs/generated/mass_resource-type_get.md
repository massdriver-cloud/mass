---
id: mass_resource-type_get.md
slug: /cli/commands/mass_resource-type_get
title: Mass Resource-Type Get
sidebar_label: Mass Resource-Type Get
---
## mass resource-type get

Get a resource type from Massdriver

### Synopsis

# Get Resource Type

Retrieves detailed information about a specific resource type, including:
- Resource type name and label
- Schema
- UI configuration
- Connection settings

## Usage

```bash
mass resource-type get <resource-type>
```

## Examples

```bash
# Get details for the "aws-s3" resource type
mass resource-type get aws-s3
```

## Options

- `--output`: Output format (text or json)


```
mass resource-type get [resource-type] [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass resource-type](/cli/commands/mass_resource-type)	 - Resource type management
