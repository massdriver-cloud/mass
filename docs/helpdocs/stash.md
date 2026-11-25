# Manage the Stash

The stash stores small values in a project, environment, component, instance, or deployment. Scripts and deployments write a value in one step and read it back in a later one.

Each scope is its own namespace. A value in an environment is not visible from its instances or deployments.

Inside a provisioner, `mass stash` signs in with the deployment's token (`MASSDRIVER_DEPLOYMENT_ID` and `MASSDRIVER_TOKEN`) when no API key is configured. That token can read and write the stash of its deployment and of that deployment's instance, environment, component, and project.

## Addresses

An address is `<scope>.<key>`. The first `.` ends the scope, so a key can hold more dots. Keys use letters, digits, and `_ . / -`.

The scope is an identifier. Its shape tells the kind:

| Scope | Kind |
| --- | --- |
| `api` | project |
| `api-prod` | environment or component |
| `api-prod-db` | instance |
| a UUID | deployment |

An environment and a component have the same shape. The CLI looks up both. When both exist, add a prefix to choose one: `env:api-db` or `component:api-db`. The prefixes `project:`, `env:`, `environment:`, `component:`, `instance:`, and `deployment:` are always accepted.

## Examples

```bash
mass stash set "$DEPLOYMENT_ID.db/hostname" "$DB_HOST"
DB_HOST=$(mass stash get "$DEPLOYMENT_ID.db/hostname")
mass stash set "$DEPLOYMENT_ID.migrations/ran" "true" --immutable
mass stash set "$DEPLOYMENT_ID.stats" "$(jq -n '{duration_s: 412, changed: 3}')" --format json
mass stash set "api-staging-db.ami-id" "ami-0abc" --ttl 86400
```
