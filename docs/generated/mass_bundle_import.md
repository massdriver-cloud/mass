---
id: mass_bundle_import.md
slug: /cli/commands/mass_bundle_import
title: Mass Bundle Import
sidebar_label: Mass Bundle Import
---
## mass bundle import

Import declared variables from IaC into massdriver.yaml params

### Synopsis

# Import IaC variables into Massdriver params

This command will scan the directories defined in the bundle steps, and attempt to identify variables declared in IaC, but not yet exposed as Massdriver params.

## Examples with Flags

By default, this command will prompt you for confirming import on every missing parameter:

```shell
mass bundle import
```

If you want to import all missing params without prompting (for bulk import or in automation), use the -a flag

```shell
mass bundle import -a
```


```
mass bundle import [flags]
```

### Options

```
  -a, --all                      Import all variables without prompting
  -b, --build-directory string   Path to a directory containing a massdriver.yaml file. (default ".")
  -h, --help                     help for import
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
