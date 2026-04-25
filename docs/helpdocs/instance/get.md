# Get Instance

Retrieves detailed information about a specific instance from Massdriver.

Your infrastructure must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

## Usage

```bash
mass instance get <instance-slug>
```

## Examples

```bash
# Get details for a VPC instance in the ecommerce production environment
mass instance get ecomm-prod-vpc
```

The instance slug can be found by hovering over the bundle in the Massdriver diagram. It follows the format: `<project-slug>-<env-slug>-<manifest-slug>`.
