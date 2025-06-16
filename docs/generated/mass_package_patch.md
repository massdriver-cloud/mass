---
id: mass_package_patch.md
slug: /cli/commands/mass_package_patch
title: Mass Package Patch
sidebar_label: Mass Package Patch
---
## mass package patch

Patch individual package parameter values

### Synopsis

# Patching package configuration on Massdriver.

Your IaC must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

Patching will perform a client-side patch of fields set on `--set`.

The `--set` argument can be called multiple times to set multiple values.

`--set` expects a JQ expression to set values.

## Examples

You can patch the package using the `slug` identifier.

The `slug` can be found by hovering over the bundle in the Massdriver diagram. The package slug is a combination of the <project-slug>-<env-slug>-<manifest-slug>

```shell
mass package patch ecomm-prod-db --set='.version = "13.4"'
```


```
mass package patch <project>-<env>-<manifest> [flags]
```

### Examples

```
mass package patch ecomm-prod-db --set='.version = "13.4"'
```

### Options

```
  -h, --help              help for patch
  -s, --set stringArray   Sets a package parameter value using JQ expressions.
```

### SEE ALSO

* [mass package](/cli/commands/mass_package)	 - Manage packages of IaC deployed in environments.
