# Build a bundle

Validates `massdriver.yaml`, dereferences its schemas, and generates IaC input declarations (such as `_massdriver_variables.tf`) for every step.

## Usage

```bash
mass bundle build [path] [flags]
```

A bundle that sets `extends: <bundle>@<constraint>` is merged with the newest matching parent and assembled in `.massdriver/build/`. See [Extending Bundles](https://docs.massdriver.cloud/bundle-development/extending-bundles).

## Flags

- `--bundle-directory, -b`: Path to a directory containing a `massdriver.yaml` file.

## Examples

```bash
# Build the bundle in the current directory
mass bundle build

# Build a bundle in another directory
mass bundle build ./bundles/payments-api
```
