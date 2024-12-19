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
      --set stringArray     set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
  -f, --values strings      specify values in a YAML file or a URL (can specify multiple)
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

