---
id: mass_infrastructure_deploy.md
slug: /cli/commands/mass_infrastructure_deploy
title: Mass Infrastructure Deploy
sidebar_label: Mass Infrastructure Deploy
---
## mass infrastructure deploy

Deploy infrastructure

### Synopsis

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


```
mass infrastructure deploy <project>-<target>-<manifest> [flags]
```

### Options

```
  -h, --help   help for deploy
```

### SEE ALSO

* [mass infrastructure](/cli/commands/mass_infrastructure)	 - Manage infrastructure
