## helm-datarobot sync

sync

### Synopsis



This command is designed to sync directly all images as part of the release manifest to a registry

Example:
```sh
$ helm datarobot sync testdata/test-chart1/ -r registry.example.com -u reg_username -p reg_password

Pulling image: docker.io/datarobot/test-image1:1.0.0
Pushing image: registry.example.com/datarobot/test-image1:1.0.0

```

```
helm-datarobot sync [flags]
```

### Options

```
  -c, --ca-cert string    Path to the custom CA certificate
  -C, --cert string       Path to the client certificate
      --dry-run           Perform a dry run without making changes
  -h, --help              help for sync
  -i, --insecure          Skip server certificate verification
  -K, --key string        Path to the client key
  -p, --password string   pass to auth
      --prefix string     append prefix on repo name
  -r, --registry string   registry to auth
  -t, --token string      pass to auth
  -u, --username string   username to auth
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

