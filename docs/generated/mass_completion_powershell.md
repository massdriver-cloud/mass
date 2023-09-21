---
id: mass_completion_powershell.md
slug: /cli/commands/mass_completion_powershell
title: Mass Completion Powershell
sidebar_label: Mass Completion Powershell
---
## mass completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	mass completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
mass completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [mass completion](/cli/commands/mass_completion)	 - Generate the autocompletion script for the specified shell
