---
id: mass_completion_zsh.md
slug: /cli/commands/mass_completion_zsh
title: Mass Completion Zsh
sidebar_label: Mass Completion Zsh
---
## mass completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(mass completion zsh)

To load completions for every new session, execute once:

#### Linux:

	mass completion zsh > "${fpath[1]}/_mass"

#### macOS:

	mass completion zsh > $(brew --prefix)/share/zsh/site-functions/_mass

You will need to start a new shell for this setup to take effect.


```
mass completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [mass completion](/cli/commands/mass_completion)	 - Generate the autocompletion script for the specified shell
