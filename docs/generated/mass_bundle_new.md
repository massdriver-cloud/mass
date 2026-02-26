---
id: mass_bundle_new.md
slug: /cli/commands/mass_bundle_new
title: Mass Bundle New
sidebar_label: Mass Bundle New
---
## mass bundle new

Create a new bundle from a template

### Synopsis

# Create a new bundle from a template

Use a local template to start building a new bundle. Templates are expected at a configured path with the structure `{templates_path}/{template}/massdriver.yaml`.

This command can run both in an interactive mode or using flags.

To get started in interactive mode run `mass bundle new` then follow the prompts.

[Massdriver documentation on building bundles](https://docs.massdriver.cloud/bundles/development)

## Configuration

Templates path can be configured in two ways (in order of precedence):

1. **Environment variable**: `MD_TEMPLATES_PATH`
2. **Config file**: `~/.config/massdriver/config.yaml`

### Config file example

```yaml
templates_path: /path/to/your/templates
```

## Template Directory Structure

Templates should be organized as:

```
templates_path/
  opentofu/
    massdriver.yaml
    src/
      main.tf
      ...
  helm-chart/
    massdriver.yaml
    chart/
      Chart.yaml
      ...
  bicep/
    massdriver.yaml
    ...
```

## Examples with Flags

Create a new bundle using an existing OpenTofu module to populate params:

```shell
mass bundle new -n foo -o massdriver -t opentofu-module -c network=massdriver/vpc -p /path/to/opentofu/dir
```

Create a new bundle using an existing Helm chart's values.yaml to populate params:

```shell
mass bundle new -n foo -o massdriver -t helm-chart -c network=massdriver/vpc -p /path/to/helm/values.yaml
```

## Skeleton massdriver.yaml Example

```yaml
schema: draft-07
name: "{{ name }}"
description: "{{ description }}"
source_url: github.com/YOUR_ORG/{{ name }}
type: bundle
access: private

steps:
  - path: src
    provisioner: opentofu

params:
  required: []
  properties: {}

connections:
  required: []
  properties: {}

artifacts:
  required: []
  properties: {}

ui:
  ui:order: []
```


```
mass bundle new [flags]
```

### Options

```
  -c, --connections strings       Connections and names to add to the bundle - example: network=massdriver/vpc
  -d, --description string        Description of the new bundle
  -h, --help                      help for new
  -n, --name string               Name of the new bundle. Setting this along with --template-name will disable the interactive prompt.
  -o, --output-directory string   Directory to output the new bundle (default ".")
  -p, --params-directory string   Path with existing params to use - opentofu module directory or helm chart values.yaml
  -t, --template-name string      Name of the bundle template to use. Setting this along with --name will disable the interactive prompt.
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
