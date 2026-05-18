# Copy Instance

Copies one instance's configuration to another instance of the same
component. The source's params (minus any fields the bundle marks
non-copyable) are written to the destination, optionally deep-merged with
`--overrides`. Deployment is a separate action — run `mass instance
deploy <destination>` when you're ready to apply.

Aliased as `promote` — same command, friendlier shape for the common
"promote staging to production" flow:

```bash
mass instance promote ecomm-staging-db --to ecomm-production-db
mass instance deploy ecomm-production-db
```

## Usage

```bash
mass instance copy <source> --to <destination> [flags]
mass instance promote <source> --to <destination> [flags]
```

## Arguments

- `source`: full identifier of the instance to copy from
  (e.g. `ecomm-staging-db`).

## Flags

- `--to`: destination instance (required). Must be built from the same
  component as the source (e.g. `ecomm-production-db`).
- `--overrides, -o`: path to a JSON or YAML file of param overrides
  deep-merged onto the source params before writing.
- `--copy-secrets`: also copy the source's secret values to the destination.
- `--copy-remote-references`: also copy the source's remote-reference
  overrides to the destination.

## Examples

```bash
# Promote staging's config to production.
mass instance promote ecomm-staging-db --to ecomm-production-db
mass instance deploy ecomm-production-db

# Promote with a size override and copy secrets.
mass instance copy ecomm-staging-db \
  --to ecomm-production-db \
  --overrides ./prod-overrides.yaml \
  --copy-secrets
```
