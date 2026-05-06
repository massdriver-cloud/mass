# Export Environment Details

Exports an environment to the local filesystem. This is useful for backups and migrations.

Data will be exported in a tiered file structure, with directory names as the environment/instance slugs:

```bash
env
├── instance1
│   ├── <data...>
├── instance2
│   ├── <data...>
```

For information about what will be exported for each instance, refer to the `mass instance export` command.

## Usage

```bash
mass environment export <project-slug>
```

## Examples

```bash
# Export the "prod" environment in the "web" project
mass environment export web-prod
```
