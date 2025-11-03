# Create Environment

Creates a new environment in a project.

## Usage

```bash
mass environment create <slug> [flags]
```

## Flags

- `--name, -n`: Environment name (defaults to slug if not provided)

## Examples

```bash
# Create an environment "dbbundle-test" (project "dbbundle" is parsed from slug)
mass environment create dbbundle-test

# Create an environment with a custom name
mass environment create dbbundle-test --name "Database Test Environment"
```
