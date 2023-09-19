# Configure applications on Massdriver.

Your application must be published as a [bundle](https://docs.massdriver.cloud/applications) to Massdriver first and be added to an environment (target).

## Examples

You can configure applications using the _fully qualified name_, its `slug`, or its ID.

The `slug` can be found by hovering over the bundle in the Massdriver diagram.

*Note:* Parameter files support bash interpolation.

**Using the fully qualified name**:

```shell
mass application configure ecomm-prod-api --params=params.json
```

**Using the slug**:

```shell
mass app cfg ecomm-prod-api-x12g -p params.json
```

**Using the ID**:

```shell
mass app cfg DC8F1D9B-BD82-4E5A-9C40-8653BC794ABC -p params.json
```
