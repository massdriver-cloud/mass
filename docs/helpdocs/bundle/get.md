# Get bundle information from Massdriver

Retrieve detailed information about a bundle, including its version, description, and source URL.

## Usage

```bash
mass bundle get <bundle-name>[@<version>] [flags]
```

If version is not specified, defaults to `latest`.

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
