## helm-datarobot load

load all images from a tgz file to a specific registry

### Synopsis



This command is designed to load all images from a tgz file to a specific registry

Example:
```sh
$ helm datarobot load images.tgz -r registry.example.com -u reg_username -p reg_password
Successfully pushed image: registry.example.com/test-image1:1.0.0

```

Authentication can be provided in various ways, including:

```sh
export REGISTRY_USERNAME=reg_username
export REGISTRY_PASSWORD=reg_password
export REGISTRY_HOST=registry.example.com
$ helm datarobot load images.tgz
```



```
helm-datarobot load [flags]
```

### Options

```
  -c, --ca-cert string           Path to the custom CA certificate
  -C, --cert string              Path to the client certificate
      --dry-run                  Perform a dry run without making changes
  -h, --help                     help for load
  -i, --insecure                 Skip server certificate verification
  -K, --key string               Path to the client key
      --output-dir string        file to save (default "export")
      --overwrite                Overwrite existing images
  -p, --password string          pass to auth
      --prefix string            append prefix on repo name
  -r, --registry string          registry to auth
      --repo string              rewrite the target repository name
      --retry-attempts int       Number of retries for pushing images (default 1)
      --retry-delay int          Delay between retries in seconds (default 5)
      --skip-image stringArray   Specify which image should be skipped (can be used multiple times)
      --suffix string            append suffix on repo name
  -t, --token string             pass to auth
  -u, --username string          username to auth
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

