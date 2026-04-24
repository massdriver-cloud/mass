---
id: mass_component_add.md
slug: /cli/commands/mass_component_add
title: Mass Component Add
sidebar_label: Mass Component Add
---
## mass component add

Add a component to a project's blueprint

### Synopsis

# Add a Component to a Project

Adds a component to a project's blueprint. The component is a design-time slot backed by the given bundle — it does not deploy anything on its own. Instances are created per environment at deploy time.

## Usage

```shell
mass component add <project-id> <bundle-oci-repo-name> --id <component-id> [--name <display-name>] [--description <description>]
```

The component ID is the final segment of all instance identifiers — for example, project `ecomm` with environment `prod` and component `db` produces instance `ecomm-prod-db`. Max 20 characters, lowercase alphanumeric only.

## Examples

```shell
# Add a Postgres bundle as "db" to the ecomm project
mass component add ecomm aws-rds-cluster --id db

# With a friendly display name and description
mass component add ecomm aws-rds-cluster --id db \
  --name "Primary Database" \
  --description "Production customer data store"
```


```
mass component add <project-id> <bundle-oci-repo-name> [flags]
```

### Examples

```
mass component add ecomm aws-rds-cluster --id db --name "Primary Database"
```

### Options

```
  -d, --description string   Optional description
  -h, --help                 help for add
      --id string            Short identifier for this component (e.g., db). Max 20 chars, lowercase alphanumeric.
  -n, --name string          Display name (defaults to --id if not provided)
```

### SEE ALSO

* [mass component](/cli/commands/mass_component)	 - Manage components in a project's blueprint
