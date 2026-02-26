---
id: mass_bundle_template_list.md
slug: /cli/commands/mass_bundle_template_list
title: Mass Bundle Template List
sidebar_label: Mass Bundle Template List
---
## mass bundle template list

List bundle templates

### Synopsis

# List Available Templates

List all available templates from your configured templates directory.

## Configuration

Templates path can be configured in two ways (in order of precedence):

1. **Environment variable**: `MD_TEMPLATES_PATH`
2. **Config file**: `~/.config/massdriver/config.yaml`

### Config file example

```yaml
templates_path: /path/to/your/templates
```

## Expected Directory Structure

Templates should be organized as `{templates_path}/{template}/massdriver.yaml`:

```
templates_path/
  opentofu/
    massdriver.yaml
    src/
      ...
  helm-chart/
    massdriver.yaml
    chart/
      ...
```

## Examples

```shell
mass bundle template list
```

## Learn More

For more information on bundle templates, see the [Bundle Templates Guide](https://docs.massdriver.cloud/guides/bundle-templates).


```
mass bundle template list [flags]
```

### Options

```
  -h, --help   help for list
```

### SEE ALSO

* [mass bundle template](/cli/commands/mass_bundle_template)	 - Application template development tools
