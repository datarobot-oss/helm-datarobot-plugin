## helm-datarobot sync

sync

### Synopsis



This command is designed to sync directly all images as part of the release manifest to a registry

Example:
```sh
$ helm datarobot sync tests/charts/test-chart1/ -r registry.example.com -u reg_username -p reg_password

Pulling image: docker.io/datarobot/test-image1:1.0.0
Pushing image: registry.example.com/datarobot/test-image1:1.0.0
```

Authentication can be provided in various ways, including:

```sh
export REGISTRY_USERNAME=reg_username
export REGISTRY_PASSWORD=reg_password
export REGISTRY_HOST=registry.example.com
$ helm datarobot sync tests/charts/test-chart1/
```


```
helm-datarobot sync [flags]
```

### Options

```
  -a, --annotation string        annotation to lookup (default "datarobot.com/images")
  -c, --ca-cert string           Path to the custom CA certificate
  -C, --cert string              Path to the client certificate
      --dry-run                  Perform a dry run without making changes
  -h, --help                     help for sync
  -i, --insecure                 Skip server certificate verification
  -K, --key string               Path to the client key
      --overwrite                Overwrite existing images
  -p, --password string          pass to auth
      --prefix string            append prefix on repo name
  -r, --registry string          registry to auth
      --repo string              rewrite the target repository name
      --retry-attempts int       Number of retries for pushing images (default 2)
      --retry-delay int          Delay between retries in seconds (default 5)
      --skip-group stringArray   Specify which image group should be skipped (can be used multiple times)
      --skip-image stringArray   Specify which image should be skipped (can be used multiple times)
      --suffix string            append suffix on repo name
  -t, --token string             pass to auth
  -u, --username string          username to auth
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

