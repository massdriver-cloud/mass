---
id: mass_infrastructure_patch.md
slug: /cli/commands/mass_infrastructure_patch
title: Mass Infrastructure Patch
sidebar_label: Mass Infrastructure Patch
---
## mass infrastructure patch

Patch individual package parameter values

### Synopsis

# Patching infrastructure configuration on Massdriver.

Your infrastructure IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be to an environment (target).

Patching will perform a client-side patch of fields set on `--set`.

The `--set` argument can be called multiple times to set multiple values.

`--set` expects a JQ expression to set values.

## Examples

You can patch infrastructure using the _fully qualified name_, its `slug`, or its ID.

The `slug` can be found by hovering over the bundle in the Massdriver diagram.

**Using the fully qualified name**:

```shell
mass infrastructure patch ecomm-prod-db --set='.version = "13.4"'
```

**Using the slug**:

```shell
mass infra patch ecomm-prod-db-x12g --set='.version = "13.4"'
```

**Using the ID**:

```shell
mass infra patch DC8F1D9B-BD82-4E5A-9C40-8653BC794ABC --set='.version = "13.4"'
```


```
mass infrastructure patch <project>-<target>-<manifest> [flags]
```

### Options

```
  -h, --help              help for patch
  -s, --set stringArray   Sets a package parameter value using JQ expressions.
```

### SEE ALSO

* [mass infrastructure](/cli/commands/mass_infrastructure)	 - Manage infrastructure
