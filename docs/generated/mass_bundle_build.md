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

An application that sets `app.runtime: <runtime>@<constraint>` is merged with the newest matching runtime and assembled in `.massdriver/build/`. See [Runtimes and Applications](https://docs.massdriver.cloud/bundle-development/runtimes-and-applications).

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
