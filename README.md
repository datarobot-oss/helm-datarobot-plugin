# helm-datarobot

[![Go Report Card](https://goreportcard.com/badge/github.com/datarobot-oss/helm-datarobot-plugin)](https://goreportcard.com/report/github.com/datarobot-oss/helm-datarobot-plugin)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/9785/badge)](https://www.bestpractices.dev/projects/9785)


The DataRobot Helm Plugin is a user-friendly tool specifically crafted to streamline image management for the DataRobot chart.

## Installation

Please make sure you have [Helm](https://helm.sh/) installed.

There are multiple ways how one can install DataRobot Helm Plugin

### Helm Plugin - install from GitHub

```sh
helm plugin install https://github.com/datarobot-oss/helm-datarobot-plugin.git
```

```sh
helm plugin update datarobot
```


### Helm Plugin - install from a local repo

Alternatively one can clone the repo directly and install the plugin from it:

```sh
helm plugin install /path/to/helm-datarobot-repo
```

The main advantages of this approach:
* non-main branch can be used
* easy to test changes when developing

### Use `helm-datarobot` binary directly

Alternatively one can just compile and use `helm-datarobot` directly:

```sh
go build -o helm-datarobot
./helm-datarobot help
```

### Managing the plugin

Command `helm plugin` supports the following subcommands:
```sh
$ helm plugin uninstall

Manage client-side Helm plugins.

Usage:
  helm plugin [command]

Available Commands:
  install     install a Helm plugin
  list        list installed Helm plugins
  uninstall   uninstall one or more Helm plugins
  update      update one or more Helm plugins

Flags:
  -h, --help   help for plugin
```

## Documentation

please check [here](./docs/helm-datarobot.md)

### Prerequisites

[Homebrew Bundle][homebrew-bundle] is the recommended tool for managing
additional tools like `go`, `gotestsum`, `helm`, etc. in local development
environment.

You need to intall [Homebrew][homebrew] on your Mac or Linux development
machine. Once it's done, please run the command below to intall all the tools:
```
brew bundle
```

Alternatively if you don't want to use Homebrew, please find the full list of
required packages in [Brewfile](./Brewfile).

[homebrew]: https://github.com/Homebrew/brew
[homebrew-bundle]: https://github.com/Homebrew/homebrew-bundle

### Building and running locally

1. Build `helm-datarobot`:
    ```
    go build
    ```
2. Run the tool:
    ```
    ./helm-datarobot help
    ```
3. One can use Helm charts located under testdata for trying the tool.

### Running tests locally

1. Run linter checks:
    ```
    golangci-lint run
    ```

2. Run unit tests:
    ```
    gotestsum
    ```


## Contributing

If you'd like to report an issue or bug, suggest improvements, or contribute code to this project, please refer to [CONTRIBUTING.md](CONTRIBUTING.md).

# Code of Conduct

This project has adopted the Contributor Covenant for its Code of Conduct.
See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) to read it in full.

# License

Licensed under the Apache License 2.0.
See [LICENSE](LICENSE) to read it in full.
