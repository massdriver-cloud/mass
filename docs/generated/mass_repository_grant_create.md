---
id: mass_repository_grant_create.md
slug: /cli/commands/mass_repository_grant_create
title: Mass Repository Grant Create
sidebar_label: Mass Repository Grant Create
---
## mass repository grant create

Share a repository with recipient projects

### Synopsis

# Create Repository Grant

Shares an OCI repository with recipient projects, so they can pull it.

Recipients are chosen by matching **project** attributes. You must say who the recipients are: either name conditions with `--condition` / `--conditions-file`, or share with the whole organization with `--all-projects`. Leaving it unstated is an error rather than a silent org-wide grant.

## Usage

```bash
mass repository grant create <name> [flags]
```

## Flags

- `--condition` — a recipient project attribute condition, `key=value`. Repeat the same key to accept a set of values; write `key=*` to accept any value as long as the attribute is set.
- `--conditions-file` — read conditions from a JSON file instead
- `--all-projects` — share with every project in the organization
- `--action` — the action to grant. Defaults to `repo:pull`, currently the only grantable repository action.

`--condition`, `--conditions-file`, and `--all-projects` are mutually exclusive.

## Examples

```bash
# Share with projects whose team attribute is platform or data
mass repository grant create aws-aurora-postgres \
  --condition team=platform \
  --condition team=data

# Share with projects that have any region attribute set
mass repository grant create aws-aurora-postgres --condition 'region=*'

# Share with every project in the organization
mass repository grant create aws-aurora-postgres --all-projects

# Read conditions from a JSON file
mass repository grant create aws-aurora-postgres --conditions-file ./conditions.json
```

## Conditions file format

One JSON object, shaped like a grant's `recipientConditions` in `mass repository grant list -o json`. Values are either `"*"` for the per-key wildcard or an array of accepted values:

```json
{
  "team": ["platform", "data"],
  "region": "*"
}
```

A file holding `null` or `{}` is rejected — use `--all-projects` to share organization-wide, so the broadest grant is always explicit.

## Notes

- Quote `key=*` so your shell does not expand the `*`.
- A comma is an ordinary character in a condition value. Unlike `--attributes`, it does not separate pairs.
- Grants are immutable; to change one, delete it and create a replacement.


```
mass repository grant create <name> [flags]
```

### Options

```
      --action string            Action to grant (default repo:pull)
      --all-projects             Share with every project in the organization
      --condition stringArray    Recipient project attribute condition; repeat a key to accept a set of values, or write key=* to accept any value
      --conditions-file string   Read recipient conditions from a JSON file
  -h, --help                     help for create
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass repository grant](/cli/commands/mass_repository_grant)	 - Manage sharing grants on an OCI repository
