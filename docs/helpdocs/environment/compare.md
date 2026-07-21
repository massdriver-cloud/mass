# Compare Environments

Performs a side-by-side comparison of two environments in the same project, instance-by-instance.

Instances are paired across environments by their underlying component. For each pair the comparison reports the bundle version diff and a flat, leaf-level diff of the params. When only one environment has an instance for a component, it is reported as present on that side only.

Environment-level attributes and default resource wiring are out of scope.

## Usage

```bash
mass environment compare <source-environment> <target-environment> [flags]
```

## Flags

- `--output, -o`: Output format, `text` (default) or `json`
- `--all`: Show unchanged params and matching instances too (by default only differences are shown)

## Examples

```bash
# Show what differs between staging and production
mass environment compare ecomm-staging ecomm-production

# Include matching instances and unchanged params
mass environment compare ecomm-staging ecomm-production --all

# Machine-readable output
mass environment compare ecomm-staging ecomm-production -o json
```
