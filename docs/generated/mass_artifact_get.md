---
id: mass_artifact_get.md
slug: /cli/commands/mass_artifact_get
title: Mass Artifact Get
sidebar_label: Mass Artifact Get
---
## mass artifact get

Get an artifact from Massdriver

### Synopsis

# Get Artifact

Retrieves detailed information about a specific artifact, including:
- Artifact ID, name, and type
- Artifact definition details
- Package information (if provisioned)
- Specs and metadata
- Available download formats

## Usage

```bash
mass artifact get <artifact-id> [flags]
```

## Examples

```bash
# Get artifact details in text format (default)
mass artifact get 12345678-1234-1234-1234-123456789012

# Get artifact details in JSON format
mass artifact get 12345678-1234-1234-1234-123456789012 --output json
```

## Options

- `--output, -o`: Output format (text or json). Defaults to text.



```
mass artifact get [artifact-id] [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass artifact](/cli/commands/mass_artifact)	 - Manage artifacts
