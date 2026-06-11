## helm-datarobot admin-chart

Generate a helm chart containing all cluster-scoped resources

### Synopsis


Extract all cluster-scoped resources (CRDs, ClusterRoles, ClusterRoleBindings,
Webhooks, etc.) from a DataRobot Helm chart and package them as a standalone
installable Helm chart.

Example:
```sh
$ helm datarobot admin-chart ./datarobot-prime-11.10.88.tgz \
    --namespace dr \
    --values my-values.yaml \
    --output ./datarobot-admin-11.10.88.tgz
```

```
helm-datarobot admin-chart [flags]
```

### Options

```
      --api-versions strings   Additional API versions for template rendering
  -d, --debug                  Print detailed resource listing
  -h, --help                   help for admin-chart
      --kube-version string    Kubernetes version for template rendering (default "v1.32.0")
      --namespace string       Kubernetes namespace (required)
      --openshift              Include OpenShift API versions (security.openshift.io/v1, route.openshift.io/v1)
  -o, --output string          Output path for the admin chart .tgz (required)
      --release-name string    Helm release name (default "datarobot")
      --set stringArray        Set values on the command line (can specify multiple)
  -f, --values strings         Specify values in a YAML file (can specify multiple)
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

