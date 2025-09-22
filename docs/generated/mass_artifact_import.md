---
id: mass_artifact_import.md
slug: /cli/commands/mass_artifact_import
title: Mass Artifact Import
sidebar_label: Mass Artifact Import
---
## mass artifact import

Import a custom artifact

### Synopsis

# Import a custom artifact

Create a custom artifact to represent infrastructure not deployed through Massdriver.

## Examples

```shell
mass artifact import -n <name> -t <type> -f <file>
```


```
mass artifact import [flags]
```

### Options

```
  -f, --file string   Artifact file
  -h, --help          help for import
  -n, --name string   Artifact name
  -t, --type string   Artifact type
```

### SEE ALSO

* [mass artifact](/cli/commands/mass_artifact)	 - Manage artifacts
