---
id: mass_resource_update.md
slug: /cli/commands/mass_resource_update
title: Mass Resource Update
sidebar_label: Mass Resource Update
---
## mass resource update

Update an imported resource

### Synopsis

# Update an imported resource

Update the payload of an imported resource. This command only works for imported resources; provisioned resources cannot be updated through the CLI.

## Examples

```shell
# Update the resource payload
mass resource update 12345678-1234-1234-1234-123456789012 -f resource.json

# Update the payload and rename the resource
mass resource update 12345678-1234-1234-1234-123456789012 -f resource.json -n new-name
```

## Options

- `--file, -f`: Path to the JSON file holding the new payload
- `--name, -n`: New name for the resource


```
mass resource update [resource-id] [flags]
```

### Options

```
  -f, --file string   Resource payload file
  -h, --help          help for update
  -n, --name string   New resource name
```

### Options inherited from parent commands

```
      --profile string   Configuration profile to use (overrides MASSDRIVER_PROFILE and the active profile)
```

### SEE ALSO

* [mass resource](/cli/commands/mass_resource)	 - Manage resources
