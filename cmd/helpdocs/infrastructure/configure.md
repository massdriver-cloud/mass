# Configure infrastructure on Massdriver.

Your infrastructure IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be to an environment (target).

Configuration will replace the full configuration of an infrastructure package in Massdriver.

## Examples

You can configure infrastructure using the _fully qualified name_, its `slug`, or its ID.

The `slug` can be found by hovering over the bundle in the Massdriver diagram.

_Note:_ Parameter files support bash interpolation.

**Using the fully qualified name**:

```shell
mass infrastructure configure ecomm-prod-vpc --params=params.json
```

**Using the slug**:

```shell
mass infra cfg ecomm-prod-vpc-x12g -p params.json
```

**Using the ID**:

```shell
mass infra cfg DC8F1D9B-BD82-4E5A-9C40-8653BC794ABC -p params.json
```
