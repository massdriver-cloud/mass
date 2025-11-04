# Create Manifest

Adds a bundle to a project as a manifest. A manifest is the context of how you plan to use a bundle in your project (e.g., a Redis bundle used for "page caching" vs "user sessions" would be two different manifests).

Manifests are added to projects and automatically created in all environments. When you configure a package (the deployed instance of a manifest), it's configured per environment.

## Usage

```bash
mass package create <slug> [flags]
```

The slug format is `project-env-manifest`, where:
- `project`: The project slug (first segment, no hyphens)
- `env`: The environment slug (second segment, no hyphens)
- `manifest`: The manifest slug (third segment, no hyphens)

## Flags

- `--name, -n`: Manifest name (defaults to manifest slug if not provided)
- `--bundle, -b`: Bundle ID or name (required)

## Examples

```bash
# Create a manifest "table" in project "test1" using bundle "aws-collab-dynamodb"
# The slug format is "test1-qa-table" where "test1" is the project, "qa" is the env, and "table" is the manifest
mass package create test1-qa-table --bundle aws-collab-dynamodb

# Create a manifest with a custom name
mass package create test1-qa-table --name "Database Table" --bundle aws-collab-dynamodb
```
