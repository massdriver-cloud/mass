# Publish a Massdriver Artifact Definition

Publish an artifact definition to Massdriver. Publishing is an upsert operation, so it will create or update an existing artifact.

## Examples

**Publish from a file**

```shell
mass definition publish -f definition.json
```

**Publish from stdin**

```shell
cat definition.json | mass definition publish -f -
```
