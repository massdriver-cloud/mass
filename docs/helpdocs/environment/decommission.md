# Decommission Environment

Tears down every instance in an environment in reverse dependency order.
The environment shell stays in place so it can be redeployed; run
`mass environment delete` afterwards to remove the empty environment.

Any in-flight environment deployment is cancelled and replaced. The
command returns as soon as the decommission wave is enqueued; instances
are torn down asynchronously.

Decommissioning is blocked when the environment has
`decommissionProtection: true`. Disable it first with
`mass environment update --decommission-protection=false` (or via the
UI / API) before retrying.

## Usage

```bash
mass environment decommission <environment>
```

## Arguments

- `environment`: full identifier of the environment to decommission
  (e.g. `ecomm-pr42`).

## Flags

- `--follow`: stream every decommission deployment's logs to stdout
  until the rollout completes. Each line is prefixed with the instance
  id so the interleaved output stays grep-friendly when multiple
  decommissions run in parallel.

## Examples

```bash
# Tear down every instance in a preview env.
mass environment decommission ecomm-pr42

# Tear it down and watch the logs.
mass environment decommission ecomm-pr42 --follow

# Full preview-env teardown: decommission instances, then delete the shell.
mass environment decommission ecomm-pr42 --follow
mass environment delete ecomm-pr42
```
