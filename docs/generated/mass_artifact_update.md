---
id: mass_artifact_update.md
slug: /cli/commands/mass_artifact_update
title: Mass Artifact Update
sidebar_label: Mass Artifact Update
---
## mass artifact update

Update an imported artifact

### Synopsis

# Update an imported artifact

Update the payload of an imported artifact. This command only works for imported artifacts; provisioned artifacts cannot be updated through the CLI.

## Examples

```shell
mass artifact update <artifact-id> -f <file>
mass artifact update <artifact-id> -f <file> -n <new-name>
```


```
mass artifact update [artifact-id] [flags]
```

### Examples

```
  # Update artifact payload
  mass artifact update 12345678-1234-1234-1234-123456789012 -f artifact.json

  # Update artifact payload and rename
  mass artifact update 12345678-1234-1234-1234-123456789012 -f artifact.json -n new-name
```

### Options

```
  -f, --file string   Artifact payload file
  -h, --help          help for update
  -n, --name string   New artifact name
```

### SEE ALSO

* [mass artifact](/cli/commands/mass_artifact)	 - Manage artifacts
