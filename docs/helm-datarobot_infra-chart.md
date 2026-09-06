## helm-datarobot infra-chart

extract all cluster-scoped resources from a chart into a standalone datarobot-infra chart

### Synopsis


Render datarobot-prime and extract all cluster-scoped resources (CRDs,
ClusterRoles, ClusterRoleBindings, webhooks, PriorityClass, GatewayClass,
APIService, ...) into a standalone datarobot-infra chart. The source chart is
read-only and never modified.

Example:
```sh
$ helm datarobot infra-chart datarobot-prime.tgz -o datarobot-infra.tgz
```

```
helm-datarobot infra-chart <prime-chart.tgz> [flags]
```

### Options

```
      --api-versions strings         extra API versions for rendering (can specify multiple)
      --cluster-read-kinds strings   "resource.group" specs for the cluster-read ClusterRole rules (e.g. storageclasses.storage.k8s.io,namespaces) (default [storageclasses.storage.k8s.io,namespaces,customresourcedefinitions.apiextensions.k8s.io])
      --crd-aggregation              generate an aggregate-to-admin/edit ClusterRole granting access to the chart's Namespaced CRD resources (cert-manager-edit pattern). Independent of --pipeline-sa. (default true)
  -d, --debug                        verbose per-resource listing
      --extra-admin-kinds strings    resource kinds to force into the infra chart even if namespaced (e.g. Role,RoleBinding,ServiceAccount)
  -h, --help                         help for infra-chart
      --keep-crds                    add helm.sh/resource-policy: keep annotation to CRDs (default true)
      --kube-version string          Helm template KubeVersion (default "v1.32.0")
      --namespace string             render namespace (.Release.Namespace) (default "datarobot")
  -o, --output string                output .tgz path (default ./datarobot-infra-<srcVersion>.tgz)
      --pipeline-sa string           name of the limited-privilege ServiceAccount to bootstrap (empty = off). Generates SA + RoleBinding to built-in admin + cluster-read ClusterRole/CRB + role-union ClusterRole/RoleBinding + cr-access ClusterRole/RoleBinding. Namespaced to --namespace at generation time.
      --release-name string          .Release.Name (default "datarobot")
      --set stringArray              set values on the command line (can specify multiple)
  -f, --values strings               specify values in a YAML file (can specify multiple)
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

