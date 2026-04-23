---
id: mass_instance_get.md
slug: /cli/commands/mass_instance_get
title: Mass Instance Get
sidebar_label: Mass Instance Get
---
## mass instance get

Get an instance

### Synopsis

# Get Instance

Retrieves detailed information about a specific instance from Massdriver.

Your infrastructure must be published as a [bundle](https://docs.massdriver.cloud/bundles) to Massdriver first and be added to an environment's canvas.

## Usage

```bash
mass instance get <instance-slug>
```

## Examples

```bash
# Get details for a VPC instance in the ecommerce production environment
mass instance get ecomm-prod-vpc
```

The instance slug can be found by hovering over the bundle in the Massdriver diagram. It follows the format: `<project-slug>-<env-slug>-<manifest-slug>`.


```
mass instance get  <project>-<env>-<manifest> [flags]
```

### Examples

```
mass instance get ecomm-prod-vpc
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
