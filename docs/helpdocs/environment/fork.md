# Fork Environment

Creates a new environment by forking an existing one. The fork is linked to
its parent via `parent_target_id`, and every instance is seeded with the
parent's params, version, and release channel.

Re-running `fork` against the same parent with the same `new-ID` resets
the existing fork's state back to the parent's — params reset, defaults
re-apply, and any `--copy-*` flags re-fire. Re-running with a *different*
parent is rejected; a fork's parent is immutable.

## Usage

```bash
mass environment fork <parent-environment> <new-ID> [flags]
```

## Arguments

- `parent-environment`: full identifier of the environment to fork from
  (e.g. `ecomm-production`).
- `new-ID`: local segment of the new environment's identifier. Must match
  `^[a-z0-9]{1,20}$` — lowercase alphanumeric only, no dashes. The full
  stored identifier becomes `<project>-<new-ID>`.

## Flags

- `--name, -n`: human-readable name (defaults to `new-ID`).
- `--description, -d`: optional environment description.
- `--attributes, -a`: custom attributes for ABAC, e.g.
  `-a region=us-east-1,data_classification=pii`.
- `--copy-environment-defaults`: also copy the parent's default resource
  connections into the fork.
- `--copy-secrets`: copy every instance's secrets from the parent into the
  fork.
- `--copy-remote-references`: copy every instance's remote references from
  the parent into the fork.

## Examples

```bash
# Stand up a staging environment as a copy of production.
mass environment fork ecomm-production staging \
  --copy-environment-defaults \
  --copy-secrets

# Re-fork to reset edits back to the parent's current state.
mass environment fork ecomm-production staging --copy-environment-defaults
```
