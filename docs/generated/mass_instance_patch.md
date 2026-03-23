---
id: mass_instance_patch.md
slug: /cli/commands/mass_instance_patch
title: Mass Instance Patch
sidebar_label: Mass Instance Patch
---
## mass instance patch

Patch individual instance parameter values

### Synopsis

# Patching package configuration on Massdriver.

Your IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

Patching will perform a client-side patch of fields set on `--set`.

The `--set` argument can be called multiple times to set multiple values.

`--set` expects a JQ expression to set values.

## Examples

You can patch the package using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the `<project-slug>-<env-slug>-<manifest-slug>`

```shell
mass package patch ecomm-prod-db --set='.version = "13.4"'
```


```
mass instance patch <project>-<env>-<manifest> [flags]
```

### Examples

```
mass instance patch ecomm-prod-db --set='.version = "13.4"'
```

### Options

```
  -h, --help              help for patch
  -s, --set stringArray   Sets an instance parameter value using JQ expressions.
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
