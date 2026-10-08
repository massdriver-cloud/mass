---
id: mass_bundle_build.md
slug: /cli/commands/mass_bundle_build
title: Mass Bundle Build
sidebar_label: Mass Bundle Build
---
## mass bundle build

Build schemas and generate IaC files from massdriver.yaml file

### Synopsis

# Build a bundle

Validates `massdriver.yaml`, dereferences its schemas, and generates IaC input declarations (such as `_massdriver_variables.tf`) for every step.

## Usage

```bash
mass bundle build [path] [flags]
```

## Extending a bundle

A bundle can reuse another bundle's IaC by naming it in `extends`. The child keeps one file; the parent's platform team keeps the code.

```yaml
extends: lambda@~1.2
name: payments-api
description: Handles card captures.
version: 0.1.0
dependencies:
  db:
    resource_type: postgres@~1
    required: true
app:
  envs:
    DB_HOST: .dependencies.db.host
  secrets:
    STRIPE_KEY:
      required: true
```

The constraint resolves to the newest matching parent version at build time (`~1.2`, `~1`, `latest`, or an exact version; append `+dev` to include development releases). The parent must be in your organization, and you need `repo:pull` on it.

The parent supplies `steps`, `params`, `ui`, `resources`, and its IaC. The child supplies:

- `name`, `version` (required), `description`, and `source_url`
- `dependencies`, added to the parent's
- `app` `envs`, `secrets`, and `policies`, added to the parent's
- `README.md`, `CHANGELOG.md`, `icon.svg`, and `operator.md`, replacing the parent's

Any other key in the child is an error, as is a dependency, env, or secret name the parent already uses. An `app.envs` expression must reference params and dependencies that exist in the merged bundle.

The merged bundle is assembled in `.massdriver/build/`, which is excluded from git. The published `massdriver.yaml` records the resolved parent:

```yaml
extends:
  name: lambda
  constraint: ~1.2
  version: 1.2.7
  digest: sha256:...
```

## Flags

- `--bundle-directory, -b`: Path to a directory containing a `massdriver.yaml` file.

## Examples

```bash
# Build the bundle in the current directory
mass bundle build

# Build a bundle in another directory
mass bundle build ./bundles/payments-api
```


```
mass bundle build [path] [flags]
```

### Options

```
  -b, --bundle-directory string   Path to a directory containing a massdriver.yaml file. (default ".")
  -h, --help                      help for build
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
