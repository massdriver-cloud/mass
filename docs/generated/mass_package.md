---
id: mass_package.md
slug: /cli/commands/mass_package
title: Mass Package
sidebar_label: Mass Package
---
## mass package

Manage packages of IaC deployed in environments.

### Synopsis

# Packages

[Packages](https://docs.massdriver.cloud/concepts/packages) are instances of infrastructure-as-code modules on your environment canvas.

Packages are used to:
- Deploy infrastructure components
- Configure application services
- Manage environment-specific settings
- Connect different components together

## Commands

- `configure`: Update package configuration
- `deploy`: Deploy a package to an environment
- `export`: Export a package to your local filesystem
- `get`: Retrieve package details and configuration
- `patch`: Update individual package parameter values


### Options

```
  -h, --help   help for package
```

### SEE ALSO

* [mass](/cli/commands/mass)	 - Massdriver Cloud CLI
* [mass package configure](/cli/commands/mass_package_configure)	 - Configure package
* [mass package deploy](/cli/commands/mass_package_deploy)	 - Deploy packages
* [mass package export](/cli/commands/mass_package_export)	 - Export packages
* [mass package get](/cli/commands/mass_package_get)	 - Get a package
* [mass package patch](/cli/commands/mass_package_patch)	 - Patch individual package parameter values
