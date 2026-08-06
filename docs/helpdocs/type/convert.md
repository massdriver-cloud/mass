# Convert Resource Type

Converts a raw JSON (or YAML) resource type schema into a `massdriver.yaml`.
Inlined instruction and export content is extracted back out into referenced
files alongside the generated `massdriver.yaml`.

A placeholder `version` is written into the output — set a real version before
publishing.

## Usage

```bash
mass resource-type convert <schema-file> [flags]
```

## Examples

```bash
# Convert a raw JSON schema, writing massdriver.yaml alongside it
mass resource-type convert ./my-resource-type.json

# Convert to a specific output path, overwriting if it exists
mass resource-type convert ./my-resource-type.json --output ./rt/massdriver.yaml --force
```
