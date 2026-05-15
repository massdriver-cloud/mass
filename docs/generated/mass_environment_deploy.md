---
id: mass_environment_deploy.md
slug: /cli/commands/mass_environment_deploy
title: Mass Environment Deploy
sidebar_label: Mass Environment Deploy
---
## mass environment deploy

Deploy every instance in an environment, in dependency order

### Synopsis

# Deploy Environment

Triggers a deployment of every instance in an environment, in dependency
order. Any in-flight environment deployment is cancelled and replaced.

The command returns as soon as the deployment is enqueued; instances are
provisioned asynchronously. Watch the deployments stream in the UI or list
them with `mass deployment list`.

## Usage

```bash
mass environment deploy <environment>
```

## Arguments

- `environment`: full identifier of the environment to deploy
  (e.g. `ecomm-staging`).

## Examples

```bash
# Deploy every instance in the staging environment.
mass environment deploy ecomm-staging

# Deploy a freshly-forked preview env.
mass environment fork ecomm-production pr42 --copy-environment-defaults
mass environment deploy ecomm-pr42
```


```
mass environment deploy [environment] [flags]
```

### Examples

```
mass environment deploy ecomm-staging
```

### Options

```
  -h, --help   help for deploy
```

### SEE ALSO

* [mass environment](/cli/commands/mass_environment)	 - Environment management
