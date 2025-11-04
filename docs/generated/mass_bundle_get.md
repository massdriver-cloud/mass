---
id: mass_bundle_get.md
slug: /cli/commands/mass_bundle_get
title: Mass Bundle Get
sidebar_label: Mass Bundle Get
---
## mass bundle get

Get bundle information from Massdriver

### Synopsis

# Get bundle information from Massdriver

Retrieve detailed information about a bundle, including its version, description, and source URL.

## Usage

```bash
mass bundle get <bundle-id>[@<version>] [flags]
```

The bundle identifier can be just the bundle ID/name, or include a version constraint:
- `<bundle-id>` - Get the latest version of the bundle
- `<bundle-id>@<version>` - Get a specific version (e.g., `1.0.0`, `~1.2`, `latest`)

## Flags

- `--output, -o`: Output format (text or json). Defaults to "text" which renders a formatted markdown document.

## Examples

```bash
# Get the latest version of a bundle
mass bundle get aws-vpc

# Get a specific version
mass bundle get aws-vpc@1.0.0

# Get output as JSON
mass bundle get aws-vpc@1.0.0 -o json

# Get output as formatted markdown (default)
mass bundle get aws-vpc@1.0.0 -o text
```



```
mass bundle get <bundle-id>[@<version>] [flags]
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
