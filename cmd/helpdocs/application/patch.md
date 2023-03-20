# Patching application configuration on Massdriver.

Your application must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be to an environment (target).

Patching will perform a client-side patch of fields set on `--set`.

The `--set` argument can be called multiple times to set multiple values.

`--set` expects a JQ expression to set values.

## Examples

You can patch applications using the _fully qualified name_, its `slug`, or its ID.

The `slug` can be found by hovering over the bundle in the Massdriver diagram.

**Using the fully qualified name**:

```shell
mass application patch ecomm-prod-db --set='.image.repository = "example/foo"'
```

**Using the slug**:

```shell
mass app patch ecomm-prod-db-x12g --set='.image.repository = "example/foo"'
```

**Using the ID**:

```shell
mass app patch DC8F1D9B-BD82-4E5A-9C40-8653BC794ABC --set='.image.repository = "example/foo"'
```
