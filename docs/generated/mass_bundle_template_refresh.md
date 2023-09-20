---
id: mass_bundle_template_refresh.md
slug: /cli/commands/mass_bundle_template_refresh
title: Mass Bundle Template Refresh
sidebar_label: Mass Bundle Template Refresh
---
## mass bundle template refresh

Update template list from the official Massdriver Github

### Synopsis

# List Available Templates.

Sync local templates cache with the [official Massdriver templates repository](https://github.com/massdriver-cloud/application-templates). Custom directories can be set for development by
setting the `MD_TEMPLATES_PATH` to your desired directory

## Examples

```shell
mass bundle template refresh
```

### With developer override

```shell
export MD_TEMPLATES_PATH=$HOME/custom/
mass bundle template refresh
```


```
mass bundle template refresh [flags]
```

### Options

```
  -h, --help   help for refresh
```

### SEE ALSO

* [mass bundle template](/cli/commands/mass_bundle_template)	 - Application template development tools
