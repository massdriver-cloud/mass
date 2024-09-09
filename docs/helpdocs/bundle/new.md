# Create a new bundle from a template

Use an existing [Application Template](https://github.com/massdriver-cloud/application-templates) to start building a new bundle.
This command can run both in an interactive mode or using flags

To get started in interactive mode run `mass bundle new` then follow the prompts

[Massdriver documentation on building bundles](https://docs.massdriver.cloud/bundles/development)

## Examples with Flags

Create a new bundle using an existing OpenTofu module to populate params:

```shell
mass bundle new -n foo -o massdriver -t opentofu-module -c network=massdriver/vpc -p /path/to/opentofu/dir
```

Create a new bundle using an existing Helm chart's values.yaml to populate params:

```shell
mass bundle new -n foo -o massdriver -t helm-chart -c network=massdriver/vpc -p /path/to/helm/values.yaml
```
