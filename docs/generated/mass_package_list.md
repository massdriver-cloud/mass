---
id: mass_package_list.md
slug: /cli/commands/mass_package_list
title: Mass Package List
sidebar_label: Mass Package List
---
## mass package list

List packages in an environment

### Synopsis

# List Packages

Lists all packages in a Massdriver environment.

## Usage

```bash
mass package list <project>-<env>
```

## Examples

```bash
# List all packages in the "ecomm" project's "prod" environment
mass package list ecomm-prod
```


```
mass package list <project>-<env> [flags]
```

### Examples

```
mass package list ecomm-prod
```

### Options

```
  -h, --help   help for list
```

### SEE ALSO

* [mass package](/cli/commands/mass_package)	 - Manage packages of IaC deployed in environments.
