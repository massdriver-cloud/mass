---
id: mass_environment_preview.md
slug: /cli/commands/mass_environment_preview
title: Mass Environment Preview
sidebar_label: Mass Environment Preview
---
## mass environment preview

Converge a preview environment from a YAML config

### Synopsis

# Preview Environment

Converges a preview environment from a YAML config: forks a base environment,
pins environment defaults, applies per-instance overrides, and triggers a deploy.

Re-running the command against the same config is safe — every step is
idempotent. Use it to ramp up a per-PR environment on every git push, and again
to reset the env back to the declared state.

## Usage

```bash
mass environment preview <ID> [flags]
```

## Arguments

- `ID`: the local segment of the preview environment's identifier (e.g.
  `pr123`). Must match `^[a-z0-9]{1,20}$` — lowercase alphanumeric only, no
  dashes. The full stored identifier becomes `<project>-<ID>`, where
  `project` comes from the config file.

## Flags

- `--file, -f`: path to the preview YAML config (default `preview.yaml`).
- `--name, -n`: human-readable environment name (defaults to `ID`).
- `--description, -d`: optional environment description.
- `--attributes, -a`: custom attributes for ABAC, e.g. `-a environment=preview,region=uswest`. Overrides `attributes:` in the config file.

## Environment-variable expansion

`${VAR}` / `$VAR` references in the config file are expanded from the
process environment before parsing. Use this for CI-injected values like PR
numbers:

```yaml
instances:
  chatsvc:
    params:
      host: chatty-pr-${GITHUB_PR}.example.com
attributes:
  pr: "${GITHUB_PR}"
```

Undefined variables expand to empty strings.

## Config schema

```yaml
# Required: the project the preview env lives in.
project: demo

# Required: the local segment of the env to fork from. The full parent
# identifier is `<project>-<baseEnvironment>`.
baseEnvironment: production

# Optional fork-level macros. Defaults to false.
copyEnvironmentDefaults: true     # carry the parent's default resources over
copySecrets: false                # fan copyInstance(copySecrets: true) to every instance
copyRemoteReferences: false       # fan copyInstance(copyRemoteReferences: true) to every instance

# Optional. Required when the organization declares attributes at the
# environment scope. Both keys and values must be strings.
attributes:
  region: us-east-1
  pr: "${GITHUB_PR}"

# Optional: pin specific resources as defaults for this env. `resourceType` is
# documentation for readers; the CLI only needs `resourceId`.
environmentDefaults:
  - resourceType: aws-iam-role
    resourceId: 161aeb95-e1c5-4f8d-803e-ef82087d7ad4

# Optional: per-instance overrides. Listed instances with no fields just
# inherit from the fork's seed.
instances:
  chatdb:
    version: "~2.0"            # stable channel

  chatsvc:
    version: "latest+dev"       # `+dev` pulls from the development channel
    params:
      ingress:
        enabled: true
        host: chatty-pr-${GITHUB_PR}.mdawssbx.com
        path: /
    secrets:
      - name: STRIPE_KEY
        value: FOO

  # listed without overrides — inherit from the fork
  imported:
  sessions:
  sessionsapi:
  sessionsfn:
  sharedvpc:
```

## Examples

```bash
# Converge a preview env for PR 123 from the default `preview.yaml`
mass environment preview pr123

# Same, with a friendly name
mass environment preview pr123 -n "Chat PR #123"

# Point at a config in another path
mass environment preview pr123 -f .github/preview.yml
```


```
mass environment preview [ID] [flags]
```

### Options

```
  -a, --attributes attributes:   Custom attributes for ABAC (e.g. -a environment=preview,region=uswest). Overrides attributes: in the config file. (default [])
  -d, --description string       Optional environment description
  -f, --file string              Path to the preview config YAML (default "preview.yaml")
  -h, --help                     help for preview
  -n, --name string              Environment name (defaults to ID if not provided)
```

### SEE ALSO

* [mass environment](/cli/commands/mass_environment)	 - Environment management
