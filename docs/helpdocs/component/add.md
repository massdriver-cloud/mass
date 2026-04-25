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
