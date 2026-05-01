---
id: mass_instance_export.md
slug: /cli/commands/mass_instance_export
title: Mass Instance Export
sidebar_label: Mass Instance Export
---
## mass instance export

Export instances

### Synopsis

# Export Instance Details

Exports an instance to the local filesystem. This is useful for backups and migrations.

Data will be exported into a directory, named via the instance slug:

```bash
instance
├── artifact_<name>.json
├── bundle
│   ├── <data...>
├── params.json
├── <path>.tfstate.json
```

The data which will be exported for each instance includes:
- **`artifact_<name>.json`**: Each resource for the deployed instance (if applicable)
- **`bundle`**: Directory containing deployed bundle version
- **`params.json`**: Current instance configuration
- **`<path>.tfstate.json`**: Terraform/OpenTofu state file for each step (if applicable)

Data will only be exported for instances in the **`PROVISIONED`** state. Data will NOT be exported for instances in the **`INITIALIZED`**, **`DECOMMISSIONED`** or **`FAILED`** state. Instances which are remote references will only download the resource files.

## Usage

```bash
mass instance export <project-slug>-<environment-slug>-<instance-slug>
```

## Examples

```bash
# Export the "app" instance in the "prod" environment of the "web" project
mass instance export web-prod-app
```


```
mass instance export <project>-<env>-<manifest> [flags]
```

### Examples

```
mass instance export ecomm-prod-vpc
```

### Options

```
  -h, --help   help for export
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
