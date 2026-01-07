# Reset Package Status

This command allows you to reset a package status back to 'Initialized'. This should only be used when a package is in an unrecoverable state - common situations include a package stuck in 'Pending' due to deployment issues, or a package that cannot be successfully decommissioned due to deployment failures.

## Examples

You can reset the package using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`

Reset and delete the deployment history:

```shell
mass package reset ecomm-prod-vpc
```
