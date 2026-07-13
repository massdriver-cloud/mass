# Deploy instances on Massdriver.

Your IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

Configuration is part of a deployment. Running `deploy` without any flags reuses the configuration of the most recent deployment.

## Examples

You can deploy using the instance ID.

The ID can be found in the details panel in the Massdriver UI. The instance ID is a combination of the `<project-id>-<env-id>-<component-id>`.

Redeploy with the same configuration as the last deployment:

```shell
mass instance deploy ecomm-prod-vpc
```

Deploy with a new full configuration. Files support bash interpolation.

```shell
mass instance deploy ecomm-prod-vpc --params=params.json
mass instance deploy ecomm-prod-vpc --params=params.tfvars
mass instance deploy ecomm-prod-vpc --params=params.yaml
mass instance deploy ecomm-prod-vpc --params=params.toml
```

Deploy with configuration read from STDIN:

```shell
echo '{"hello": "world"}' | mass instance deploy ecomm-prod-vpc --params=-
```

Copy configuration between environments:

```shell
mass instance get api-prod-web -o json | jq .params | mass instance deploy api-staging-web --params=-
```

Patch the last deployed configuration with one or more JQ expressions:

```shell
mass instance deploy ecomm-prod-db --patch='.version = "13.4"'
mass instance deploy ecomm-prod-db --patch='.version = "13.4"' --patch='.size = "large"'
```

Run a dry-run plan to preview changes without provisioning. `--plan` combines with `--params`/`--patch` to preview a proposed configuration, and with `--follow` to stream the plan output:

```shell
mass instance deploy ecomm-prod-db --plan
mass instance deploy ecomm-prod-db --plan --patch='.version = "13.4"' --follow
```

Propose a deployment for approval instead of running it immediately. The deployment is created in `PROPOSED` status and runs only once approved with `mass deployment approve` (or discarded with `mass deployment reject`). `--plan` and `--propose` cannot be combined:

```shell
mass instance deploy ecomm-prod-db --propose --message "bump db to 13.4" --patch='.version = "13.4"'
```
