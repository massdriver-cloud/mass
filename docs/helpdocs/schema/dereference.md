# Dereferences a JSON Schema Document

This command will expand all the `$ref` statements in a JSON Schema. This command is useful when managing resource type schemas and using `$refs` to keep your schemas "DRY".

## Examples

From an existing file

```shell
mass schema dereference --file resource-type.json
```

From stdin

```shell
cat resource-type.json | mass schema dereference -f -
```
