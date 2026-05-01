---
id: mass_instance_deploy.md
slug: /cli/commands/mass_instance_deploy
title: Mass Instance Deploy
sidebar_label: Mass Instance Deploy
---
## mass instance deploy

Deploy instances

### Synopsis

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


```
mass instance deploy <project>-<env>-<manifest> [flags]
```

### Examples

```
mass instance deploy ecomm-prod-vpc
```

### Options

```
  -h, --help                help for deploy
  -m, --message string      Add a message when deploying
  -p, --params string       Path to params json, tfvars or yaml file. Use '-' to read from stdin. When provided, the full configuration is replaced. Supports bash interpolation.
  -P, --patch stringArray   Patch the last deployed configuration using a JQ expression. Can be specified multiple times.
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
