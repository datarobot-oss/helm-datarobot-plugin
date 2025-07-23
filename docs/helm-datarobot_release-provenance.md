## helm-datarobot release-provenance

Show image provenance (repo and commit) for all images in the chart

### Synopsis


The "release-provenance" subcommand inspects all images declared in the chart`s "datarobot.com/images"
annotation (and its subcharts), and attempts to extract provenance information for each image. Provenance
typically includes the source repository and the commit SHA or tag from which the image was built.

Example:

```sh
$ helm-datarobot release-provenance datarobot-prime-11.0.0.tgz
[
  {
    "image": "docker.io/datarobotdev/test-service1:1.2.3",
    "repo": "test-repo1",
    "commit": "123abc456def7890123456789abcdef012345678"
  },
  {
    "image": "docker.io/datarobotdev/test-service2:4.5.6",
    "repo": "test-repo2",
    "commit": "abcdef1234567890abcdef1234567890abcdef12"
  },
  {
    "image": "docker.io/datarobotdev/test-service3:7.8.9",
    "repo": "test-repo3",
    "commit": "fedcba9876543210fedcba9876543210fedcba98"
  }
]
```


```
helm-datarobot release-provenance [flags]
```

### Options

```
  -h, --help   help for release-provenance
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

