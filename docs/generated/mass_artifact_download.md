---
id: mass_artifact_download.md
slug: /cli/commands/mass_artifact_download
title: Mass Artifact Download
sidebar_label: Mass Artifact Download
---
## mass artifact download

Download an artifact in the specified format

### Synopsis

# Download Artifact

Downloads an artifact in the specified format. The artifact data is rendered according to the artifact definition's schema and returned in the requested format.

## Usage

```bash
mass artifact download <artifact-id> [flags]
```

## Examples

```bash
# Download artifact in JSON format (default)
mass artifact download 12345678-1234-1234-1234-123456789012

# Download artifact in YAML format
mass artifact download 12345678-1234-1234-1234-123456789012 --format yaml

# Download artifact in JSON format explicitly
mass artifact download 12345678-1234-1234-1234-123456789012 --format json
```

## Options

- `--format, -f`: Download format (json, yaml, etc.). Defaults to json.

## Notes

- The available formats depend on the artifact definition's configuration
- Use `mass artifact get` to see available formats for a specific artifact



```
mass artifact download [artifact-id] [flags]
```

### Options

```
  -f, --format string   Download format (json, yaml, etc.) (default "json")
  -h, --help            help for download
```

### SEE ALSO

* [mass artifact](/cli/commands/mass_artifact)	 - Manage artifacts
