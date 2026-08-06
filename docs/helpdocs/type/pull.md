# Pull Resource Type

Pulls a published resource type from your organization's catalog into a local
directory.

## Usage

```bash
mass resource-type pull <resource-type> [flags]
```

## Examples

```bash
# Pull the latest version into a directory named after the resource type
mass resource-type pull my-resource-type

# Pull a specific version into a specific directory
mass resource-type pull my-resource-type --version 1.2.0 --directory ./out
```
