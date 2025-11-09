# Get Package

Retrieves detailed information about a specific package from Massdriver.

Your infrastructure must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

## Usage

```bash
mass package get <package-slug> [flags]
```

## Flags

- `-o, --output`: Output format (text or json). Defaults to text (markdown).

## Examples

```bash
# Get details for a VPC package in the ecommerce production environment
mass package get ecomm-prod-vpc

# Get package as JSON and extract just the params using jq
mass package get ecomm-prod-vpc -o json | jq .params

# Get package status from JSON output
mass package get ecomm-prod-vpc -o json | jq .status

# Get package environment details
mass package get ecomm-prod-vpc -o json | jq .environment

# Save package configuration to a file
mass package get ecomm-prod-vpc -o json > package.json
```

The package slug can be found by hovering over the bundle in the Massdriver diagram. It follows the format: `<project-slug>-<env-slug>-<manifest-slug>`.
