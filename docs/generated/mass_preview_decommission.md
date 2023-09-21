---
id: mass_preview_decommission.md
slug: /cli/commands/mass_preview_decommission
title: Mass Preview Decommission
sidebar_label: Mass Preview Decommission
---
## mass preview decommission

Decommissions a preview environment in your project

### Synopsis

# Decommission a Preview Environment

Decommissions a prevew environment in your project.

**Example:**

Decommission an environment (legacy term: target) named `pr11` from `ecomm`:

```shell
mass preview decommission ecomm-pr11
```


```
mass preview decommission $projectTargetSlug [flags]
```

### Options

```
  -h, --help   help for decommission
```

### SEE ALSO

* [mass preview](/cli/commands/mass_preview)	 - Create & deploy preview environments
