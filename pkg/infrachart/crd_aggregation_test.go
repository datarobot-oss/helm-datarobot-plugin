package infrachart

import (
	"strings"
	"testing"
)

func TestBuildCRDAggregation_Labels(t *testing.T) {
	crdResources := map[string][]string{
		"example.com": {"foos", "bars"},
	}
	rs, err := BuildCRDAggregation(crdResources, "myrelease")
	if err != nil {
		t.Fatalf("BuildCRDAggregation error: %v", err)
	}
	if len(rs) != 1 {
		t.Fatalf("want 1 resource, got %d", len(rs))
	}
	m := unmarshalMap(t, rs[0].RawYAML)
	meta, ok := m["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata missing or wrong type")
	}
	labels, ok := meta["labels"].(map[string]interface{})
	if !ok {
		t.Fatal("labels missing or wrong type")
	}
	if labels["rbac.authorization.k8s.io/aggregate-to-admin"] != "true" {
		t.Errorf("aggregate-to-admin label missing or wrong: %v", labels)
	}
	if labels["rbac.authorization.k8s.io/aggregate-to-edit"] != "true" {
		t.Errorf("aggregate-to-edit label missing or wrong: %v", labels)
	}
}

func TestBuildCRDAggregation_PerGroupRules(t *testing.T) {
	crdResources := map[string][]string{
		"example.com":      {"foos", "bars"},
		"other.example.io": {"widgets"},
	}
	rs, err := BuildCRDAggregation(crdResources, "myrelease")
	if err != nil {
		t.Fatalf("BuildCRDAggregation error: %v", err)
	}
	if len(rs) != 1 {
		t.Fatalf("want 1 resource, got %d", len(rs))
	}
	m := unmarshalMap(t, rs[0].RawYAML)
	rulesRaw, ok := m["rules"].([]interface{})
	if !ok || len(rulesRaw) != 2 {
		t.Fatalf("want 2 rules (one per group), got %v", m["rules"])
	}

	// Groups are sorted: example.com < other.example.io
	r0 := rulesRaw[0].(map[string]interface{})
	assertStringSlice(t, r0, "apiGroups", []string{"example.com"})
	assertStringSlice(t, r0, "resources", []string{"foos", "bars"})
	assertStringSlice(t, r0, "verbs", []string{"*"})

	r1 := rulesRaw[1].(map[string]interface{})
	assertStringSlice(t, r1, "apiGroups", []string{"other.example.io"})
	assertStringSlice(t, r1, "resources", []string{"widgets"})
}

func TestBuildCRDAggregation_DeterministicGroupOrder(t *testing.T) {
	// Run twice with reversed insertion order; result must be identical.
	a := map[string][]string{
		"aaa.io": {"alphas"},
		"zzz.io": {"zetas"},
	}
	b := map[string][]string{
		"zzz.io": {"zetas"},
		"aaa.io": {"alphas"},
	}
	rsA, err := BuildCRDAggregation(a, "rel")
	if err != nil {
		t.Fatal(err)
	}
	rsB, err := BuildCRDAggregation(b, "rel")
	if err != nil {
		t.Fatal(err)
	}
	if rsA[0].RawYAML != rsB[0].RawYAML {
		t.Errorf("non-deterministic output:\nA: %s\nB: %s", rsA[0].RawYAML, rsB[0].RawYAML)
	}
	// First group in output must be aaa.io (sorted ascending)
	if !strings.Contains(rsA[0].RawYAML, "aaa.io") {
		t.Errorf("expected aaa.io first in sorted output:\n%s", rsA[0].RawYAML)
	}
}

func TestBuildCRDAggregation_EmptyMap(t *testing.T) {
	rs, err := BuildCRDAggregation(map[string][]string{}, "myrelease")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs != nil {
		t.Errorf("want nil, got %v", rs)
	}
}

func TestBuildCRDAggregation_EmptyReleaseName(t *testing.T) {
	_, err := BuildCRDAggregation(map[string][]string{"example.com": {"foos"}}, "")
	if err == nil {
		t.Fatal("expected error for empty releaseName, got nil")
	}
	if !strings.Contains(err.Error(), "releaseName") {
		t.Errorf("error should mention releaseName, got: %v", err)
	}
}

func TestBuildCRDAggregation_ReleaseName(t *testing.T) {
	rs, err := BuildCRDAggregation(map[string][]string{"example.com": {"foos"}}, "myrelease")
	if err != nil {
		t.Fatal(err)
	}
	if rs[0].Name != "myrelease-crd-edit-aggregate" {
		t.Errorf("name = %q, want myrelease-crd-edit-aggregate", rs[0].Name)
	}
}
