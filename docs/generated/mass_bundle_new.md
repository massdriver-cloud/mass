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

Use an existing [Application Template](https://github.com/massdriver-cloud/application-templates) to start building a new bundle.
This command can run both in an interactive mode or using flags

To get started in interactive mode run `mass bundle new` then follow the prompts

[Massdriver documentation on building bundles](https://docs.massdriver.cloud/bundles/development)

## Examples with Flags

Create a new bundle using an existing Terraform module to populate params:

```shell
mass bundle new -n foo -o massdriver -t terraform-module -c network=massdriver/vpc -p /path/to/terraform/dir
```

Create a new bundle using an existing Helm chart's values.yaml to populate params:

```shell
mass bundle new -n foo -o massdriver -t helm-chart -c network=massdriver/vpc -p /path/to/helm/values.yaml
```


```
mass bundle new [flags]
```

### Options

```
  -c, --connections strings       Connections and names to add to the bundle - example: network=massdriver/vpc
  -d, --description string        Description of the new bundle
  -h, --help                      help for new
  -n, --name string               Name of the new bundle
  -o, --output-directory string   Directory to output the new bundle (default ".")
  -p, --params-directory string   Path with existing params to use - terraform module directory or helm chart values.yaml
  -t, --template-name string      Name of the bundle template to use
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
