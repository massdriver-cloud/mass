---
id: mass_completion_fish.md
slug: /cli/commands/mass_completion_fish
title: Mass Completion Fish
sidebar_label: Mass Completion Fish
---
## mass completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	mass completion fish | source

To load completions for every new session, execute once:

	mass completion fish > ~/.config/fish/completions/mass.fish

You will need to start a new shell for this setup to take effect.


```
mass completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [mass completion](/cli/commands/mass_completion)	 - Generate the autocompletion script for the specified shell
