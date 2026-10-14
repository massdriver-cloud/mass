---
id: mass_bundle_publish.md
slug: /cli/commands/mass_bundle_publish
title: Mass Bundle Publish
sidebar_label: Mass Bundle Publish
---
## mass bundle publish

Publish bundle to Massdriver's package manager

### Synopsis

# Publish a bundle

Builds, lints, and publishes a bundle to your organization's catalog. The bundle's repository must already exist (see `mass bundle create`).

## Usage

```bash
mass bundle publish [path] [flags]
```

A bundle that sets `extends` is merged with the newest matching parent and published as a single bundle. See [Extending Bundles](https://docs.massdriver.cloud/bundle-development/extending-bundles).

## Flags

- `--bundle-directory, -b`: Path to a directory containing a `massdriver.yaml` file.
- `--development, -d`: Publish as a development release (`<version>-dev.<timestamp>`).
- `--fail-warnings, -f`: Fail on warnings from the linter.
- `--skip-lint, -s`: Skip linting.

## Examples

```bash
# Publish the bundle in the current directory
mass bundle publish

# Publish a development release
mass bundle publish --development
```


```
mass bundle publish [path] [flags]
```

### Options

```
  -b, --bundle-directory string   Path to a directory containing a massdriver.yaml file. (default ".")
  -d, --development               Publish the bundle as a development release.
  -f, --fail-warnings             Fail on warnings from the linter
  -h, --help                      help for publish
  -s, --skip-lint                 Skip linting
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
