# Publish Artifact Definition

Publishes a new or updated artifact definition to Massdriver.

## Usage

```bash
mass definition publish --file <definition-file>
```

## Examples

```bash
# Publish a definition from a file
mass definition publish --file my-definition.json

# Publish a definition from stdin
cat my-definition.json | mass definition publish --file -
```

## Options

- `--file`: Path to the definition file (use - for stdin)
