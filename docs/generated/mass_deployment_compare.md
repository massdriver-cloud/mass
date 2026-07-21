---
id: mass_deployment_compare.md
slug: /cli/commands/mass_deployment_compare
title: Mass Deployment Compare
sidebar_label: Mass Deployment Compare
---
## mass deployment compare

Compare two deployments' bundle version and params

### Synopsis

# Compare Deployments

Diffs two deployments' snapshotted configuration: the bundle version on each side plus a flat, leaf-level diff of the params.

Use it to audit what a deploy changed ("what did this deployment do to the params?") or to contrast deploys from different points in time. Runtime state, logs, and produced artifacts are out of scope.

The two deployments need not target the same instance, though comparing unrelated instances naturally reports every param as present on one side only.

## Usage

```bash
mass deployment compare <source-deployment-id> <target-deployment-id> [flags]
```

## Flags

- `--output, -o`: Output format, `text` (default) or `json`
- `--all`: Show unchanged params too (by default only differing params are shown)

## Examples

```bash
# Show only the params that changed between two deployments
mass deployment compare 1111... 2222...

# Show every param, changed or not
mass deployment compare 1111... 2222... --all

# Machine-readable output
mass deployment compare 1111... 2222... -o json
```


```
mass deployment compare <source-deployment-id> <target-deployment-id> [flags]
```

### Examples

```
mass deployment compare 1111... 2222...
```

### Options

```
      --all             Show unchanged params too (default shows only differences)
  -h, --help            help for compare
  -o, --output string   Output format (text or json) (default "text")
```

### SEE ALSO

* [mass deployment](/cli/commands/mass_deployment)	 - Manage deployments
