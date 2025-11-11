# Configure infrastructure on Massdriver.

Your IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

This command will replace the full configuration of an infrastructure package in Massdriver.

## Examples

You can configure the package using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`

_Note:_ Parameter files support bash interpolation.

Configure from file:

```shell
mass package configure ecomm-prod-vpc --params=params.json
mass package configure ecomm-prod-vpc --params=params.tfvars
mass package configure ecomm-prod-vpc --params=params.yaml
mass package configure ecomm-prod-vpc --params=params.toml
```

Configure from STDIN:

```shell
echo '{"hello": "world"}' | mass package configure ecomm-prod-vpc --params=-
```

Copy configuration between environments:

```shell
mass pkg get api-prod-web -o json | jq .params | mass pkg cfg api-staging-web --params -
```

Copy configuration and change some values:
```shell
mass pkg get api-prod-web -o json \
  | jq '.params.domain = "staging.example.com"' \
  | jq '.params.image.tag = "latest"' \
  | mass pkg cfg api-staging-web --params -
```
