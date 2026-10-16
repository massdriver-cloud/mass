# Publish a bundle

Builds, lints, and publishes a bundle to your organization's catalog. The bundle's repository must already exist (see `mass bundle create`).

## Usage

```bash
mass bundle publish [path] [flags]
```

An application that sets `extends_runtime` is merged with the newest matching runtime and published as a single bundle. See [Runtimes and Applications](https://docs.massdriver.cloud/bundle-development/runtimes-and-applications).

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
