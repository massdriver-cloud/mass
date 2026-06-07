---
id: mass_instance_list.md
slug: /cli/commands/mass_instance_list
title: Mass Instance List
sidebar_label: Mass Instance List
---
## mass instance list

List instances in an environment

### Synopsis

# List Instances

Lists all instances in a Massdriver environment.

## Usage

```bash
mass instance list <project>-<env>
```

## Examples

```bash
# List all instances in the "ecomm" project's "prod" environment
mass instance list ecomm-prod
```


```
mass instance list <project>-<env> [flags]
```

### Examples

```
mass instance list ecomm-prod
```

### Options

```
      --bundle string   Filter by bundle version (name@version) or release channel (name@latest)
  -h, --help            help for list
  -o, --output string   Output format (table, json) (default "table")
      --repo string     Filter by OCI repo name (matches all versions of a bundle)
      --status string   Filter by lifecycle status (initialized, provisioned, decommissioned, failed)
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
