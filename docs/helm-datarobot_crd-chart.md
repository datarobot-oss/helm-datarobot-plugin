## helm-datarobot crd-chart

extract CRDs from a chart into a standalone datarobot-infra chart

### Synopsis



Render a chart (e.g. datarobot-prime) and extract only its
CustomResourceDefinitions into a standalone, installable datarobot-infra chart.

Example:
```sh
$ helm datarobot crd-chart datarobot-prime.tgz -o datarobot-infra.tgz
```

```
helm-datarobot crd-chart <prime-chart.tgz> [flags]
```

### Options

```
      --api-versions strings   extra API versions for rendering (can specify multiple)
  -d, --debug                  verbose per-CRD listing
  -h, --help                   help for crd-chart
      --keep-crds              add helm.sh/resource-policy: keep annotation (default true)
      --kube-version string    Helm template KubeVersion (default "v1.32.0")
      --namespace string       render namespace (.Release.Namespace) (default "datarobot")
  -o, --output string          output .tgz path (default ./datarobot-infra-<srcVersion>.tgz)
      --release-name string    .Release.Name (default "datarobot")
      --set stringArray        set values on the command line (can specify multiple)
  -f, --values strings         specify values in a YAML file (can specify multiple)
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

