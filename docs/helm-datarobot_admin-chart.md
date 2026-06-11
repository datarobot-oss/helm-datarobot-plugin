## helm-datarobot admin-chart

Generate a helm chart containing all cluster-scoped resources

### Synopsis


Extract all cluster-scoped resources (CRDs, ClusterRoles, ClusterRoleBindings,
Webhooks, etc.) from a DataRobot Helm chart and package them as a standalone
installable Helm chart.

Use --extra-admin-kinds to force additional kinds (e.g. Role, RoleBinding,
ServiceAccount) into the admin chart for PNC-style restricted RBAC environments
where cluster-admin install handles those privileged resources.

Namespace pinning: when --pipeline-sa is set, the generated pipeline RBAC
(ServiceAccount, RoleBindings) is namespaced to the --namespace value supplied
at GENERATION time. The generated admin chart must be installed into that exact
namespace, and that namespace must equal the namespace used for the future
application chart install. Changing --namespace after generation requires
regenerating the admin chart.

Example:
```sh
$ helm datarobot admin-chart ./datarobot-prime-11.10.88.tgz \
    --namespace dr \
    --values my-values.yaml \
    --keep-crds=true \
    --extra-admin-kinds Role,RoleBinding,ServiceAccount
```

```
helm-datarobot admin-chart [flags]
```

### Options

```
      --api-versions strings         Additional API versions for template rendering
      --cluster-read-kinds strings   "resource.group" specs for the cluster-read ClusterRole rules (e.g. storageclasses.storage.k8s.io,namespaces). (default [storageclasses.storage.k8s.io,namespaces,customresourcedefinitions.apiextensions.k8s.io])
  -d, --debug                        Print detailed resource listing
      --extra-admin-kinds strings    Resource kinds to force into the admin chart even if namespaced (e.g. Role,RoleBinding,ServiceAccount). Useful for PNC-style restricted RBAC where cluster-admin cannot be assumed.
  -h, --help                         help for admin-chart
      --keep-crds                    Add helm.sh/resource-policy: keep annotation to CRDs in the generated chart to prevent CR data loss on uninstall. (default true)
      --kube-version string          Kubernetes version for template rendering (default "v1.32.0")
      --namespace string             Kubernetes namespace for .Release.Namespace template rendering (default: "default"). When --pipeline-sa is set, this value is also used as the namespace for the generated SA and RoleBindings; it must equal the namespace where both the admin chart and the future app chart will be installed.
      --openshift                    Include OpenShift API versions (security.openshift.io/v1, route.openshift.io/v1)
  -o, --output string                Output path for the admin chart .tgz (default: ./datarobot-admin-<version>.tgz)
      --pipeline-sa string           Name of the limited-privilege ServiceAccount to bootstrap for pipeline use. When set, generates: SA + RoleBinding to built-in admin ClusterRole + cluster-read ClusterRole/CRB (for chart render-time lookup calls) + role-union ClusterRole/RoleBinding (RBAC escalation prevention fix). Empty = feature off. Note: the SA and RoleBindings are namespaced to the --namespace value at GENERATION time; the admin chart must be installed into that namespace and it must equal the future app-install namespace. Note: role-union covers rules from kind: Role only; RoleBindings referencing ClusterRoles are not covered yet.
      --release-name string          Helm release name (default "datarobot")
      --set stringArray              Set values on the command line (can specify multiple)
  -f, --values strings               Specify values in a YAML file (can specify multiple)
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

