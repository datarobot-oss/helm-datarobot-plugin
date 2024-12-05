## helm-datarobot validate

validate

### Synopsis



This command is designed to validate all images presnet in a chart are declared inside the annotation

Example:
```sh
$ helm datarobot validate chart.tgz

```

```
helm-datarobot validate [flags]
```

### Options

```
  -a, --annotation string   annotation to lookup (default "datarobot.com/images")
  -d, --debug               debug
  -h, --help                help for validate
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

