---
id: mass_resource_grant_create.md
slug: /cli/commands/mass_resource_grant_create
title: Mass Resource Grant Create
sidebar_label: Mass Resource Grant Create
---
## mass resource grant create

Share a resource with recipient environments

### Synopsis

# Create Resource Grant

Shares a resource with recipient environments, so they can export it.

Recipients are chosen by matching **environment** attributes. You must say who the recipients are: either name conditions with `--condition` / `--conditions-file`, or share with the whole organization with `--all-environments`. Leaving it unstated is an error rather than a silent org-wide grant.

## Usage

```bash
mass resource grant create <resource-id> [flags]
```

## Flags

- `--condition` — a recipient environment attribute condition, `key=value`. Repeat the same key to accept a set of values; write `key=*` to accept any value as long as the attribute is set.
- `--conditions-file` — read conditions from a JSON file instead
- `--all-environments` — share with every environment in the organization
- `--action` — the action to grant. Defaults to `resource:export`, currently the only grantable resource action.

`--condition`, `--conditions-file`, and `--all-environments` are mutually exclusive.

## Examples

```bash
# Share with environments whose stage attribute is dev or staging
mass resource grant create api-prod-database-connection \
  --condition stage=dev \
  --condition stage=staging

# Share with environments that have any region attribute set
mass resource grant create api-prod-database-connection --condition 'region=*'

# Share with every environment in the organization
mass resource grant create api-prod-database-connection --all-environments

# Read conditions from a JSON file
mass resource grant create api-prod-database-connection --conditions-file ./conditions.json
```

## Conditions file format

One JSON object, shaped like a grant's `recipientConditions` in `mass resource grant list -o json`. Values are either `"*"` for the per-key wildcard or an array of accepted values:

```json
{
  "stage": ["dev", "staging"],
  "region": "*"
}
```

A file holding `null` or `{}` is rejected — use `--all-environments` to share organization-wide, so the broadest grant is always explicit.

## Notes

- The `<resource-id>` is a UUID for imported resources, or a friendly slug for provisioned ones.
- Quote `key=*` so your shell does not expand the `*`.
- A comma is an ordinary character in a condition value. Unlike `--attributes`, it does not separate pairs.
- Grants are immutable; to change one, delete it and create a replacement.


```
mass resource grant create <resource-id> [flags]
```

### Options

```
      --action string            Action to grant (default resource:export)
      --all-environments         Share with every environment in the organization
      --condition stringArray    Recipient environment attribute condition; repeat a key to accept a set of values, or write key=* to accept any value
      --conditions-file string   Read recipient conditions from a JSON file
  -h, --help                     help for create
```

### SEE ALSO

* [mass resource grant](/cli/commands/mass_resource_grant)	 - Manage sharing grants on a resource
