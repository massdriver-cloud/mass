---
id: mass_resource-type_convert.md
slug: /cli/commands/mass_resource-type_convert
title: Mass Resource-Type Convert
sidebar_label: Mass Resource-Type Convert
---
## mass resource-type convert

Convert a raw JSON schema resource type into a massdriver.yaml

### Synopsis

# Convert Resource Type

Converts a raw JSON (or YAML) resource type schema into a `massdriver.yaml`.
Inlined instruction and export content is extracted back out into referenced
files alongside the generated `massdriver.yaml`.

A placeholder `version` is written into the output — set a real version before
publishing.

## Usage

```bash
mass resource-type convert <schema-file> [flags]
```

## Examples

```bash
# Convert a raw JSON schema, writing massdriver.yaml alongside it
mass resource-type convert ./my-resource-type.json

# Convert to a specific output path, overwriting if it exists
mass resource-type convert ./my-resource-type.json --output ./rt/massdriver.yaml --force
```


```
mass resource-type convert <schema-file> [flags]
```

### Options

```
  -f, --force           Overwrite existing files
  -h, --help            help for convert
  -o, --output string   Path to write the massdriver.yaml (default: alongside the input file)
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass resource-type](/cli/commands/mass_resource-type)	 - Resource type management
