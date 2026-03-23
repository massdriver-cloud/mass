---
id: mass_instance_deploy.md
slug: /cli/commands/mass_instance_deploy
title: Mass Instance Deploy
sidebar_label: Mass Instance Deploy
---
## mass instance deploy

Deploy instances

### Synopsis

# Deploy packages on Massdriver.

Your IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

## Examples

You can deploy the package using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`

```shell
mass package deploy ecomm-prod-vpc
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
  -h, --help             help for deploy
  -m, --message string   Add a message when deploying
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
