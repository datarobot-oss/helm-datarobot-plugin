## helm-datarobot load

load

### Synopsis



This command is designed to load all images from a tgz file to a specific registry

Example:
```sh
$ helm datarobot load images.tgz -r registry.example.com -u reg_username -p reg_password
Successfully pushed image: registry.example.com/test-image1:1.0.0

```

```
helm-datarobot load [flags]
```

### Options

```
  -h, --help              help for load
  -p, --password string   pass to auth
      --prefix string     append prefix on repo name
  -r, --registry string   registry to auth
      --skip-tls-verify   Ignore SSL certificate verification (optional)
  -t, --token string      pass to auth
  -u, --username string   username to auth
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

