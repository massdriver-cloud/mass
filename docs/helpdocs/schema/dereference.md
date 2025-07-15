# Dereferences a JSON Schema Document

This command will expand all the `$ref` statements in a JSON Schema. This command is useful when managing artifact definition schemas and using `$refs` to keep your definitions "DRY".

## Examples

From a existing file

```shell
mass schema dereference --file artdef.json
```

From stdin

```shell
cat artdef.json | mass schema dereference -f -
```
