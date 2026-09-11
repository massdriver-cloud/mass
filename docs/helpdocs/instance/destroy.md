# Destroy Instance

Destroys (decommissions) an instance. This permanently deletes the instance and all the resources it provisioned.

## Usage

```bash
mass instance destroy <project>-<env>-<manifest> [flags]
```

## Flags

- `-m, --message` — add a message when decommissioning
- `-f, --force` — skip the confirmation prompt
- `-p, --params` — path to a params json, tfvars or yaml file; `-` reads stdin
- `-P, --patch` — patch the last deployed configuration with a JQ expression (repeatable)
- `--follow` — stream the deployment's logs to stdout until it completes

## Examples

```bash
# Destroy an instance, confirming at the prompt
mass instance destroy api-prod-db

# Skip the prompt, for scripted teardown
mass instance destroy api-prod-db --force

# Destroy and watch the logs until it finishes
mass instance destroy api-prod-db --force --follow
```
