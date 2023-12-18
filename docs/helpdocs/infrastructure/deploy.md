# Deploy infrastructure on Massdriver.

Your infrastructure IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be configured for a given environment (target).

## Examples

You can deploy infrastructure using the _fully qualified name_ of the application or its `slug`.

The `slug` can be found by hovering over the bundle in the Massdriver diagram.

**Using the fully qualified name**:

```shell
mass infra deploy ecomm-prod-vpc
```

**Using the slug**:

```shell
mass infra deploy ecomm-prod-vpc-x12g
```
