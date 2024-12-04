# helm-datarobot

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


## Local development environment

`pod-metering-agent` is a [Golang][golang] project. The project is implemented
as a binary built from [main.go](./main.go).

[golang]: https://go.dev/

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

