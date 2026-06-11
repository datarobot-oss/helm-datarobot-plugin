## helm-datarobot filter-resources

Filter rendered Helm manifests by scope (stdin → stdout)

### Synopsis


Read rendered Kubernetes manifests from stdin and write only the requested
partition to stdout. Use as a Helm post-renderer to strip cluster-scoped
resources before installing with limited privileges, or to extract only the
admin resources.

  --keep app   (default) write namespaced resources — for limited-privilege install
  --keep admin           write cluster-scoped resources only

Example:
```sh
$ helm install datarobot ./datarobot-prime-11.10.88.tgz \
    --post-renderer ./helm-datarobot \
    --post-renderer-args filter-resources \
    --post-renderer-args --keep=app
```

Note: --post-renderer-args requires Helm >= 3.5.
The binary is built at repo root via `make build`.


```
helm-datarobot filter-resources [flags]
```

### Options

```
      --extra-admin-kinds strings   Resource kinds to treat as admin (cluster-scoped) even if namespaced
  -h, --help                        help for filter-resources
      --keep string                 Partition to keep: "app" (namespaced resources) or "admin" (cluster-scoped resources) (default "app")
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

