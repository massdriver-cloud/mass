# Copy Instance

Copies one instance's configuration into another instance of the same
component. The source's params (minus any fields the bundle marks
non-copyable) are written to the destination, optionally deep-merged with
`--overrides`. A plan deployment is created on the destination so the
changes can be reviewed before applying.

Aliased as `promote` — same command, friendlier shape for the common
"promote staging to production" flow:

```bash
mass instance promote ecomm-staging-db ecomm-production-db
```

## Usage

```bash
mass instance copy <source> <destination> [flags]
mass instance promote <source> <destination> [flags]
```

## Arguments

- `source`: full identifier of the instance to copy from
  (e.g. `ecomm-staging-db`).
- `destination`: full identifier of the instance to copy into
  (e.g. `ecomm-production-db`). Must be built from the same component as
  the source.

## Flags

- `--message, -m`: optional message attached to the plan deployment created
  on the destination (think: commit message).
- `--overrides, -o`: path to a JSON or YAML file of param overrides
  deep-merged onto the source params before writing.
- `--copy-secrets`: also copy the source's secret values to the destination.
- `--copy-remote-references`: also copy the source's remote-reference
  overrides to the destination.

## Examples

```bash
# Promote staging's config to production (review the plan before applying).
mass instance promote ecomm-staging-db ecomm-production-db -m "Promote DB config"

# Promote with a size override and copy secrets.
mass instance copy ecomm-staging-db ecomm-production-db \
  --overrides ./prod-overrides.yaml \
  --copy-secrets \
  -m "Scale up DB for production"
```
