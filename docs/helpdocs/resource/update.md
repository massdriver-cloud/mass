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
