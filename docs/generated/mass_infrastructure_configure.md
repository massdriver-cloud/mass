---
id: mass_infrastructure_configure.md
slug: /cli/commands/mass_infrastructure_configure
title: Mass Infrastructure Configure
sidebar_label: Mass Infrastructure Configure
---
## mass infrastructure configure

Configure infrastructure

### Synopsis

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


```
mass infrastructure configure <project>-<target>-<manifest> [flags]
```

### Options

```
  -h, --help            help for configure
  -p, --params string   Path to params JSON file. This file supports bash interpolation. (default "./params.json")
```

### SEE ALSO

* [mass infrastructure](/cli/commands/mass_infrastructure)	 - Manage infrastructure
