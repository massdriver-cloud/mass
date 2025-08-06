# Export Environment Details

Exports an environment to the local filesystem. This is useful for backups and migrations.

Data will be exported in a teired file structure, with directory names as the environment/package slugs:

```bash
env
├── package1
│   ├── <data...>
├── package2
│   ├── <data...>
```

For information about what will be exported for each package, refer to the `mass package export` command.

## Usage

```bash
mass environment export <project-slug>
```

## Examples

```bash
# Export the "prod" environment in the "web" project
mass environment export web-prod
```
