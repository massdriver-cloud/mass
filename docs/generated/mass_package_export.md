---
id: mass_package_export.md
slug: /cli/commands/mass_package_export
title: Mass Package Export
sidebar_label: Mass Package Export
---
## mass package export

Export packages

### Synopsis

# Export Package Details

Exports a package to the local filesystem. This is useful for backups and migrations.

Data will be exported into a directory, named via the package slug:

```bash
package
├── artifact_<name>.json
├── bundle
│   ├── <data...>
├── params.json
├── <path>.tfstate.json
```

The data which will be exported for each package includes:
- **`artifact_<name>.json`**: Each artifact for the deploy package (if applicable)
- **`bundle`**: Directory containing deployed bundle version
- **`params.json`**: Current package configuration
- **`\<path\>.tfstate.json`**: Terraform/OpenTofu state file for each step (if applicable)

Data will only be exported for packages in the **`RUNNING`** state. Data will NOT be exported for packages in the **`INITIALIZED`**, **`DECOMMISSIONED`** or **`FAILED`** state. Packages which are remote references will only download the artifacts files.

## Usage

```bash
mass package export <project-slug>-<environment-slug>-<package-slug>
```

## Examples

```bash
# Export the "app" package in the "prod" environment of the "web" project
mass package export web-prod-app
```


```
mass package export <project>-<env>-<manifest> [flags]
```

### Examples

```
mass package export ecomm-prod-vpc
```

### Options

```
  -h, --help   help for export
```

### SEE ALSO

* [mass package](/cli/commands/mass_package)	 - Manage packages of IaC deployed in environments.
