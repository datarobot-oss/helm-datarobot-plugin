package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	skipDepUpdate, _ := os.LookupEnv("HELM_DATAROBOT_TEST_SKIP_DEPENDENCY_UPDATE")
	if strings.ToLower(skipDepUpdate) != "true" {
		charts := []string{
			"testdata/test-chart3",
			"testdata/test-chart2",
			"testdata/test-chart1",
		}
		for _, chart := range charts {
			fmt.Printf("Running `helm dependency update %s`", chart)
			cmd := exec.Command("helm", "dependency", "update", chart)
			err := cmd.Run()
			if err != nil {
				fmt.Printf("Failed to run `helm dependency update %s`", chart)
				os.Exit(1)
			}
		}
	}

	// Now run the tests
	os.Exit(m.Run())
}
