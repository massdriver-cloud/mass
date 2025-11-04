# Create Project

Creates a new project in Massdriver.

## Usage

```bash
mass project create <slug> [flags]
```

## Flags

- `--name, -n`: Project name (defaults to slug if not provided)

## Examples

```bash
# Create a project with slug "dbbundle"
mass project create dbbundle

# Create a project with a custom name
mass project create dbbundle --name "Database Bundle Project"
```
