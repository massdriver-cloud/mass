# Reset Instance Status

This command allows you to reset an instance status back to 'Initialized'. This should only be used when an instance is in an unrecoverable state - common situations include an instance stuck in 'Pending' due to deployment issues, or an instance that cannot be successfully decommissioned due to deployment failures.

## Examples

You can reset the instance using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The instance slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`

Reset and delete the deployment history:

```shell
mass instance reset ecomm-prod-vpc
```
