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
mass resource update <resource-id> -f <file>
mass resource update <resource-id> -f <file> -n <new-name>
```


```
mass resource update [resource-id] [flags]
```

### Examples

```
  # Update resource payload
  mass resource update 12345678-1234-1234-1234-123456789012 -f resource.json

  # Update resource payload and rename
  mass resource update 12345678-1234-1234-1234-123456789012 -f resource.json -n new-name
```

### Options

```
  -f, --file string   Resource payload file
  -h, --help          help for update
  -n, --name string   New resource name
```

### SEE ALSO

* [mass resource](/cli/commands/mass_resource)	 - Manage resources
