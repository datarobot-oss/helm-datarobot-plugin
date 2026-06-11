package adminchart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func makeResource(kind, name string) manifest.Resource {
	return manifest.Resource{
		Kind: kind,
		Name: name,
		RawYAML: strings.TrimSpace(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ` + kind + `
metadata:
  name: ` + name),
	}
}

func TestBuildChart(t *testing.T) {
	resources := []manifest.Resource{
		makeResource("CustomResourceDefinition", "foo.example.com"),
		makeResource("ClusterRole", "role-a"),
		makeResource("ClusterRole", "role-b"),
	}

	opts := ChartOptions{Name: "admin", Version: "1.0.0", AppVersion: "2.0.0"}
	c, err := BuildChart(resources, opts)
	if err != nil {
		t.Fatalf("BuildChart error: %v", err)
	}

	if c.Metadata.Name != "admin" {
		t.Errorf("name = %q, want admin", c.Metadata.Name)
	}
	if c.Metadata.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", c.Metadata.Version)
	}
	if c.Metadata.AppVersion != "2.0.0" {
		t.Errorf("appVersion = %q, want 2.0.0", c.Metadata.AppVersion)
	}
	if c.Metadata.APIVersion != "v2" {
		t.Errorf("apiVersion = %q, want v2", c.Metadata.APIVersion)
	}
	if c.Metadata.Type != "application" {
		t.Errorf("type = %q, want application", c.Metadata.Type)
	}

	nameSet := make(map[string]bool)
	for _, tpl := range c.Templates {
		nameSet[tpl.Name] = true
	}
	if !nameSet["templates/clusterroles.yaml"] {
		t.Error("missing templates/clusterroles.yaml")
	}
	if !nameSet["templates/customresourcedefinitions.yaml"] {
		t.Error("missing templates/customresourcedefinitions.yaml")
	}
	if len(c.Templates) != 2 {
		t.Errorf("template count = %d, want 2", len(c.Templates))
	}
}

func TestBuildChart_EmptyResources(t *testing.T) {
	_, err := BuildChart(nil, ChartOptions{Name: "x", Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected error on empty resources, got nil")
	}
}

func TestBuildChart_MultiDocPerKind(t *testing.T) {
	resources := []manifest.Resource{
		makeResource("ClusterRole", "role-a"),
		makeResource("ClusterRole", "role-b"),
	}

	c, err := BuildChart(resources, ChartOptions{Name: "x", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("BuildChart error: %v", err)
	}
	if len(c.Templates) != 1 {
		t.Fatalf("template count = %d, want 1", len(c.Templates))
	}
	content := string(c.Templates[0].Data)
	if !strings.Contains(content, "\n---\n") {
		t.Errorf("expected --- separator between docs, got:\n%s", content)
	}
	if !strings.Contains(content, "role-a") || !strings.Contains(content, "role-b") {
		t.Errorf("expected both role-a and role-b in template")
	}
}

func TestPackageChart(t *testing.T) {
	resources := []manifest.Resource{
		makeResource("ClusterRole", "role-a"),
	}
	c, err := BuildChart(resources, ChartOptions{Name: "myadmin", Version: "0.1.0", AppVersion: "1.0.0"})
	if err != nil {
		t.Fatalf("BuildChart error: %v", err)
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "myadmin-0.1.0.tgz")

	if err := PackageChart(c, outPath); err != nil {
		t.Fatalf("PackageChart error: %v", err)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file not found: %v", err)
	}

	loaded, err := loader.Load(outPath)
	if err != nil {
		t.Fatalf("loader.Load error: %v", err)
	}
	if loaded.Metadata.Name != "myadmin" {
		t.Errorf("loaded name = %q, want myadmin", loaded.Metadata.Name)
	}
}
