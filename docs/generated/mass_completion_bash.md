## mass completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(mass completion bash)

To load completions for every new session, execute once:

#### Linux:

	mass completion bash > /etc/bash_completion.d/mass

#### macOS:

	mass completion bash > $(brew --prefix)/etc/bash_completion.d/mass

You will need to start a new shell for this setup to take effect.


```
mass completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [mass completion](mass_completion.md)	 - Generate the autocompletion script for the specified shell

###### Auto generated by spf13/cobra on 19-Sep-2023