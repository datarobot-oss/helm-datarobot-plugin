package main

import (
	"github.com/datarobot-oss/helm-datarobot-plugin/cmd"
)

var (
	version = "0.0.0"
	commit  = "localdev"
)

func main() {
	cmd.SetVersionInfo(version, commit)
	cmd.Execute()
}
