## helm-datarobot save

save images in single tgz file

### Synopsis



This command is designed to save all images as part of the release manifest in single tgz file

Example:
```sh
$ helm datarobot save tests/charts/test-chart1/
Pulling image: docker.io/datarobot/test-image1:1.0.0
....
Pulling image: docker.io/datarobot/test-image2:2.0.0
....
Tarball created successfully: images.tgz
$ du -h images.tgz
14M    images.tgz

```

```
helm-datarobot save [flags]
```

### Options

```
  -a, --annotation string   annotation to lookup (default "datarobot.com/images")
      --dry-run             Perform a dry run without making changes
  -h, --help                help for save
  -o, --output string       file to save (default "images.tgz")
```

### SEE ALSO

* [helm-datarobot](helm-datarobot.md)	 - datarobot helm plugin

