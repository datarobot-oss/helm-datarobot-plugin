## helm-datarobot generate

generate

### Synopsis



This command is designed to extract all images and generate the image document annotations from a given change

Example:
```sh
$ helm datarobot generate chart.tgz

```

```
helm-datarobot generate [flags]
```

### Options

```
  -a, --annotation string   annotation to lookup (default "datarobot.com/images")
  -d, --debug               debug
  -h, --help                help for generate
      --set stringArray     set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
  -f, --values strings      specify values in a YAML file or a URL (can specify multiple)
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

