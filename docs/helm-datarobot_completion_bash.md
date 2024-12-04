## helm-datarobot completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(helm-datarobot completion bash)

To load completions for every new session, execute once:

#### Linux:

	helm-datarobot completion bash > /etc/bash_completion.d/helm-datarobot

#### macOS:

	helm-datarobot completion bash > $(brew --prefix)/etc/bash_completion.d/helm-datarobot

You will need to start a new shell for this setup to take effect.


```
helm-datarobot completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [helm-datarobot completion](helm-datarobot_completion.md)	 - Generate the autocompletion script for the specified shell

