# Get Package

Retrieves detailed information about a specific package from Massdriver.

Your infrastructure must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

## Usage

```bash
mass package get <package-slug>
```

## Examples

```bash
# Get details for a VPC package in the ecommerce production environment
mass package get ecomm-prod-vpc
```

The package slug can be found by hovering over the bundle in the Massdriver diagram. It follows the format: `<project-slug>-<env-slug>-<manifest-slug>`.
