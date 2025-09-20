# Export Project Details

Exports a project to the local filesystem. This is useful for backups and migrations.

Data will be exported in a teired file structure, with directory names as the project/environment/package slugs:

```bash
project
├── env1
│   ├── package1
│   │   ├── <data...>
│   ├── package2
│   │   ├── <data...>
├── env2
│   ├── package1
│   │   ├── <data...>
│   ├── package2
│   │   ├── <data...>
```

For information about what will be exported for each environment, refer to the `mass environment export` command.
For information about what will be exported for each package, refer to the `mass package export` command.

## Usage

```bash
mass project export <project-slug>
```

## Examples

```bash
# Export the "web" project
mass project export web
```
