## helm-datarobot release-manifest

release-manifest

### Synopsis


Subcommand `release-manifest` is conceptually similar to subcommand `images`.
it supports more than 1 chart, so we can produce a single manifest and other umbrella charts.

Example:
```sh
$ helm datarobot release-manifest testdata/test-chart1/
images:
	test-image1.tar.zst:
	source: docker.io/datarobotdev/test-image1:1.0.0
	name: docker.io/datarobot/test-image1
	internal_dockerhub_name: docker.io/datarobotdev/test-image1
	tag: 1.0.0
	test-image2.tar.zst:
	source: docker.io/datarobotdev/test-image2:2.0.0
	name: docker.io/datarobot/test-image2
	internal_dockerhub_name: docker.io/datarobotdev/test-image2
	tag: 2.0.0
	test-image3.tar.zst:
	source: docker.io/datarobotdev/test-image3:3.0.0
	name: docker.io/datarobot/test-image3
	internal_dockerhub_name: docker.io/datarobotdev/test-image3
	tag: 3.0.0
```

```

```
helm-datarobot release-manifest [flags]
```

### Options

```
  -a, --annotation string   annotation to lookup (default "datarobot.com/images")
  -h, --help                help for release-manifest
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

