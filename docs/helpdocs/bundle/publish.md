# Publish a bundle

Builds, lints, and publishes a bundle to your organization's catalog. The bundle's repository must already exist (see `mass bundle create`).

## Usage

```bash
mass bundle publish [path] [flags]
```

A bundle that sets `extends` is merged with the newest parent version matching its constraint and published as a single bundle containing the parent's IaC. See `mass bundle build --help`.

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
