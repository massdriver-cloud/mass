# Get Package

Gets a package's details from Massdriver.

Your IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

## Examples

You can get package details using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the <project-slug>-<env-slug>-<manifest-slug>

```shell
mass package get ecomm-prod-vpc
```
