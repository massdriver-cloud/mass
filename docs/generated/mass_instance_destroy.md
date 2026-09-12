---
id: mass_instance_destroy.md
slug: /cli/commands/mass_instance_destroy
title: Mass Instance Destroy
sidebar_label: Mass Instance Destroy
---
## mass instance destroy

Destroy (decommission) an instance

### Synopsis

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


```
mass instance destroy <project>-<env>-<manifest> [flags]
```

### Options

```
      --follow              Stream the deployment's logs to stdout until it completes
  -f, --force               Skip confirmation prompt
  -h, --help                help for destroy
  -m, --message string      Add a message when decommissioning
  -p, --params string       Path to params json, tfvars or yaml file. Use '-' to read from stdin. When provided, the full configuration is replaced. Supports bash interpolation.
  -P, --patch stringArray   Patch the last deployed configuration using a JQ expression. Can be specified multiple times.
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass instance](/cli/commands/mass_instance)	 - Manage instances of IaC deployed in environments.
