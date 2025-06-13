# Deploy packages on Massdriver.

Your IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

## Examples

You can deploy the package using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the <project-slug>-<env-slug>-<manifest-slug>

```shell
mass package deploy ecomm-prod-vpc
```
