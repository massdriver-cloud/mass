---
id: mass_bundle_pull.md
slug: /cli/commands/mass_bundle_pull
title: Mass Bundle Pull
sidebar_label: Mass Bundle Pull
---
## mass bundle pull

Pull bundle from Massdriver to local directory

```
mass bundle pull <bundle-name> [flags]
```

### Options

```
  -d, --directory string   Directory to output the bundle. Defaults to bundle name.
  -f, --force              Force pull even if the directory already exists. This will overwrite existing files.
  -h, --help               help for pull
  -t, --tag string         Bundle tag (defaults to 'latest') (default "latest")
```

### SEE ALSO

* [mass bundle](/cli/commands/mass_bundle)	 - Generate and publish bundles
