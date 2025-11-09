# Configure infrastructure on Massdriver.

Your IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

This command will replace the full configuration of an infrastructure package in Massdriver.

## Usage

```bash
mass package configure <package-slug> [flags]
```

## Flags

- `-p, --params`: Path to params JSON, tfvars, or YAML file. Use `-` to read from stdin. Defaults to `./params.json`. This file supports bash interpolation.

## Examples

You can configure the package using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`

_Note:_ Parameter files support bash interpolation.

```bash
# Configure package with params file
mass package configure ecomm-prod-vpc --params=params.json

# Configure package with params file using short flag
mass package configure ecomm-prod-vpc -p params.json

# Configure package by reading params from stdin
mass package configure ecomm-prod-vpc --params=-

# Configure package with tfvars file
mass package configure ecomm-prod-vpc --params=terraform.tfvars

# Configure package with YAML file
mass package configure ecomm-prod-vpc --params=params.yaml

# Pipe params from another command
echo '{"version": "1.0.0"}' | mass package configure ecomm-prod-vpc --params=-

# Clone configurations between environments
mass package get ecomm-staging-vpc -o json | jq .params | mass package cfg ecomm-dev-vpc --params -
```
