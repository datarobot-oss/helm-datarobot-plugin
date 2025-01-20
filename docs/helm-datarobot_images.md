## helm-datarobot images

list images from a given chart

### Synopsis


DataRobot introduced a custom annotation `datarobot.com/images` to solve
this problem. This annotation lets chart developers manifest which images are
required by the application. Those images will be included into enterprise
releases automatically.

Example:
```sh
$ yq ".annotations" tests/charts/test-chart1/Chart.yaml
datarobot.com/images: |
- name: test-image1
image: docker.io/datarobotdev/test-image1:{{.Chart.AppVersion}}
```

The value of `datarobot.com/images` annotation is a template (pay attention to
`|`) that is going to be rendered with `gotpl` (just like everything else in
Helm) with `.Chart` available in the context.

Subcommand `images` parses, combines and returns `datarobot.com/images`
annotations of a chart and its subcharts, e.g.:

```sh
$ helm datarobot images tests/charts/test-chart1
- name: test-image1
image: docker.io/datarobotdev/test-image1:1.0.0
- name: test-image2
image: docker.io/datarobotdev/test-image2:2.0.0
- name: test-image3
image: docker.io/datarobotdev/test-image3:3.0.0
```

```

```
helm-datarobot images [flags]
```

### Options

```
  -a, --annotation string   annotation to lookup (default "datarobot.com/images")
  -h, --help                help for images
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

