package adminchart

import (
	"strings"
	"testing"

	sigsyaml "sigs.k8s.io/yaml"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/manifest"
)

// ---- ClusterReadRules -------------------------------------------------------

func mustClusterReadRules(t *testing.T, kinds []string) []map[string]interface{} {
	t.Helper()
	rules, err := ClusterReadRules(kinds)
	if err != nil {
		t.Fatalf("ClusterReadRules(%v) unexpected error: %v", kinds, err)
	}
	return rules
}

func TestClusterReadRules_Dotted(t *testing.T) {
	rules := mustClusterReadRules(t, []string{"storageclasses.storage.k8s.io"})
	if len(rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rules))
	}
	r := rules[0]
	assertStringSlice(t, r, "apiGroups", []string{"storage.k8s.io"})
	assertStringSlice(t, r, "resources", []string{"storageclasses"})
	assertStringSlice(t, r, "verbs", []string{"get", "list", "watch"})
}

func TestClusterReadRules_CoreGroup(t *testing.T) {
	rules := mustClusterReadRules(t, []string{"namespaces"})
	if len(rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rules))
	}
	r := rules[0]
	assertStringSlice(t, r, "apiGroups", []string{""})
	assertStringSlice(t, r, "resources", []string{"namespaces"})
	assertStringSlice(t, r, "verbs", []string{"get", "list", "watch"})
}

func TestClusterReadRules_Multiple(t *testing.T) {
	rules := mustClusterReadRules(t, []string{
		"storageclasses.storage.k8s.io",
		"namespaces",
		"customresourcedefinitions.apiextensions.k8s.io",
	})
	if len(rules) != 3 {
		t.Fatalf("want 3 rules, got %d", len(rules))
	}
	assertStringSlice(t, rules[0], "apiGroups", []string{"storage.k8s.io"})
	assertStringSlice(t, rules[0], "resources", []string{"storageclasses"})
	assertStringSlice(t, rules[1], "apiGroups", []string{""})
	assertStringSlice(t, rules[1], "resources", []string{"namespaces"})
	assertStringSlice(t, rules[2], "apiGroups", []string{"apiextensions.k8s.io"})
	assertStringSlice(t, rules[2], "resources", []string{"customresourcedefinitions"})
}

func TestClusterReadRules_FirstDotOnly(t *testing.T) {
	// "customresourcedefinitions.apiextensions.k8s.io" — split on FIRST dot only
	rules := mustClusterReadRules(t, []string{"customresourcedefinitions.apiextensions.k8s.io"})
	r := rules[0]
	assertStringSlice(t, r, "apiGroups", []string{"apiextensions.k8s.io"})
	assertStringSlice(t, r, "resources", []string{"customresourcedefinitions"})
}

func TestClusterReadRules_SkipsEmptyEntries(t *testing.T) {
	rules, err := ClusterReadRules([]string{"namespaces", "  ", "", "storageclasses.storage.k8s.io"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("want 2 rules (blank entries skipped), got %d", len(rules))
	}
}

func TestClusterReadRules_EmptyResourcePartIsError(t *testing.T) {
	_, err := ClusterReadRules([]string{".somegroup"})
	if err == nil {
		t.Fatal("expected error for entry with empty resource part, got nil")
	}
	if !strings.Contains(err.Error(), ".somegroup") {
		t.Errorf("error should mention bad entry, got: %v", err)
	}
}

// ---- BuildPipelineRBAC validation -------------------------------------------

func TestBuildPipelineRBAC_ValidationErrors(t *testing.T) {
	base := PipelineRBACOptions{
		SAName:           "pipeline-sa",
		Namespace:        "default",
		ReleaseName:      "myrelease",
		ClusterReadRules: mustClusterReadRules(t, DefaultClusterReadKinds),
	}

	tests := []struct {
		name string
		mod  func(*PipelineRBACOptions)
	}{
		{"empty SAName", func(o *PipelineRBACOptions) { o.SAName = "" }},
		{"empty Namespace", func(o *PipelineRBACOptions) { o.Namespace = "" }},
		{"empty ReleaseName", func(o *PipelineRBACOptions) { o.ReleaseName = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := base
			tc.mod(&opts)
			_, err := BuildPipelineRBAC(opts)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// ---- BuildPipelineRBAC full (6 resources) -----------------------------------

func TestBuildPipelineRBAC_Full(t *testing.T) {
	unionRules := []map[string]interface{}{
		{
			"apiGroups": []string{"apps"},
			"resources": []string{"deployments"},
			"verbs":     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
		},
	}
	opts := PipelineRBACOptions{
		SAName:           "pipeline-sa",
		Namespace:        "dr",
		ReleaseName:      "myrelease",
		ClusterReadRules: mustClusterReadRules(t, DefaultClusterReadKinds),
		UnionRules:       unionRules,
	}

	resources, err := BuildPipelineRBAC(opts)
	if err != nil {
		t.Fatalf("BuildPipelineRBAC error: %v", err)
	}
	if len(resources) != 6 {
		t.Fatalf("want 6 resources, got %d: %v", len(resources), kindNames(resources))
	}

	// 1. ServiceAccount
	sa := resources[0]
	if sa.Kind != "ServiceAccount" {
		t.Errorf("[0] kind = %q, want ServiceAccount", sa.Kind)
	}
	if sa.Name != "pipeline-sa" {
		t.Errorf("[0] name = %q, want pipeline-sa", sa.Name)
	}
	if sa.Namespace != "dr" {
		t.Errorf("[0] namespace = %q, want dr", sa.Namespace)
	}
	saMap := unmarshalMap(t, sa.RawYAML)
	assertNestedString(t, saMap, "dr", "metadata", "namespace")

	// 2. RoleBinding pipeline-admin
	rb1 := resources[1]
	if rb1.Kind != "RoleBinding" {
		t.Errorf("[1] kind = %q, want RoleBinding", rb1.Kind)
	}
	if rb1.Name != "myrelease-pipeline-admin" {
		t.Errorf("[1] name = %q, want myrelease-pipeline-admin", rb1.Name)
	}
	if rb1.Namespace != "dr" {
		t.Errorf("[1] namespace = %q, want dr", rb1.Namespace)
	}
	rb1Map := unmarshalMap(t, rb1.RawYAML)
	assertNestedString(t, rb1Map, "ClusterRole", "roleRef", "kind")
	assertNestedString(t, rb1Map, "admin", "roleRef", "name")
	assertSubjectSA(t, rb1Map, "pipeline-sa", "dr")

	// 3. ClusterRole cluster-read
	cr := resources[2]
	if cr.Kind != "ClusterRole" {
		t.Errorf("[2] kind = %q, want ClusterRole", cr.Kind)
	}
	if cr.Name != "myrelease-pipeline-cluster-read" {
		t.Errorf("[2] name = %q, want myrelease-pipeline-cluster-read", cr.Name)
	}
	if cr.Namespace != "" {
		t.Errorf("[2] namespace = %q, want empty", cr.Namespace)
	}

	// 4. ClusterRoleBinding cluster-read
	crb := resources[3]
	if crb.Kind != "ClusterRoleBinding" {
		t.Errorf("[3] kind = %q, want ClusterRoleBinding", crb.Kind)
	}
	if crb.Name != "myrelease-pipeline-cluster-read" {
		t.Errorf("[3] name = %q, want myrelease-pipeline-cluster-read", crb.Name)
	}
	crbMap := unmarshalMap(t, crb.RawYAML)
	assertNestedString(t, crbMap, "myrelease-pipeline-cluster-read", "roleRef", "name")
	assertSubjectSA(t, crbMap, "pipeline-sa", "dr")

	// 5. ClusterRole role-union
	crUnion := resources[4]
	if crUnion.Kind != "ClusterRole" {
		t.Errorf("[4] kind = %q, want ClusterRole", crUnion.Kind)
	}
	if crUnion.Name != "myrelease-pipeline-role-union" {
		t.Errorf("[4] name = %q, want myrelease-pipeline-role-union", crUnion.Name)
	}

	// 6. RoleBinding role-union
	rb2 := resources[5]
	if rb2.Kind != "RoleBinding" {
		t.Errorf("[5] kind = %q, want RoleBinding", rb2.Kind)
	}
	if rb2.Name != "myrelease-pipeline-role-union" {
		t.Errorf("[5] name = %q, want myrelease-pipeline-role-union", rb2.Name)
	}
	if rb2.Namespace != "dr" {
		t.Errorf("[5] namespace = %q, want dr", rb2.Namespace)
	}
	rb2Map := unmarshalMap(t, rb2.RawYAML)
	assertNestedString(t, rb2Map, "ClusterRole", "roleRef", "kind")
	assertNestedString(t, rb2Map, "myrelease-pipeline-role-union", "roleRef", "name")
	assertSubjectSA(t, rb2Map, "pipeline-sa", "dr")
}

// ---- BuildPipelineRBAC empty UnionRules → 4 resources -----------------------

func TestBuildPipelineRBAC_NoUnion(t *testing.T) {
	opts := PipelineRBACOptions{
		SAName:           "pipeline-sa",
		Namespace:        "dr",
		ReleaseName:      "myrelease",
		ClusterReadRules: mustClusterReadRules(t, DefaultClusterReadKinds),
		UnionRules:       nil,
	}
	resources, err := BuildPipelineRBAC(opts)
	if err != nil {
		t.Fatalf("BuildPipelineRBAC error: %v", err)
	}
	if len(resources) != 4 {
		t.Fatalf("want 4 resources, got %d: %v", len(resources), kindNames(resources))
	}
	for _, r := range resources {
		if strings.Contains(r.Name, "role-union") {
			t.Errorf("unexpected role-union resource %q when UnionRules empty", r.Name)
		}
	}
}

// ---- BuildChart with PipelineRBAC ------------------------------------------

func TestBuildChart_WithPipelineRBAC(t *testing.T) {
	resources := []manifest.Resource{
		makeResource("ClusterRole", "role-a"),
	}
	rbacResources, err := BuildPipelineRBAC(PipelineRBACOptions{
		SAName:           "pipeline-sa",
		Namespace:        "dr",
		ReleaseName:      "myrelease",
		ClusterReadRules: mustClusterReadRules(t, DefaultClusterReadKinds),
	})
	if err != nil {
		t.Fatalf("BuildPipelineRBAC error: %v", err)
	}

	opts := ChartOptions{
		Name:         "admin",
		Version:      "1.0.0",
		PipelineRBAC: rbacResources,
	}
	c, err := BuildChart(resources, opts)
	if err != nil {
		t.Fatalf("BuildChart error: %v", err)
	}

	var rbacTpl []byte
	for _, tpl := range c.Templates {
		if tpl.Name == "templates/pipeline-rbac.yaml" {
			rbacTpl = tpl.Data
		}
	}
	if rbacTpl == nil {
		t.Fatal("templates/pipeline-rbac.yaml not found in chart")
	}

	content := string(rbacTpl)
	// All 4 resources (no union) should be present.
	for _, needle := range []string{"ServiceAccount", "RoleBinding", "ClusterRole", "ClusterRoleBinding"} {
		if !strings.Contains(content, needle) {
			t.Errorf("pipeline-rbac.yaml missing %q", needle)
		}
	}
	// Docs joined with separator.
	if !strings.Contains(content, "---") {
		t.Errorf("expected --- separator between docs")
	}
}

func TestBuildChart_NoPipelineRBAC(t *testing.T) {
	resources := []manifest.Resource{
		makeResource("ClusterRole", "role-a"),
	}
	c, err := BuildChart(resources, ChartOptions{Name: "admin", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("BuildChart error: %v", err)
	}
	for _, tpl := range c.Templates {
		if tpl.Name == "templates/pipeline-rbac.yaml" {
			t.Error("templates/pipeline-rbac.yaml unexpectedly present when PipelineRBAC empty")
		}
	}
}

// ---- helpers ----------------------------------------------------------------

func kindNames(rs []manifest.Resource) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Kind + "/" + r.Name
	}
	return out
}

func unmarshalMap(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := sigsyaml.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal: %v\nYAML:\n%s", err, raw)
	}
	return m
}

func assertStringSlice(t *testing.T, rule map[string]interface{}, key string, want []string) {
	t.Helper()
	raw, ok := rule[key]
	if !ok {
		t.Errorf("rule missing key %q", key)
		return
	}
	var got []string
	switch v := raw.(type) {
	case []string:
		got = v
	case []interface{}:
		for _, item := range v {
			got = append(got, item.(string))
		}
	default:
		t.Errorf("key %q unexpected type %T", key, raw)
		return
	}
	if len(got) != len(want) {
		t.Errorf("key %q: got %v, want %v", key, got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("key %q[%d]: got %q, want %q", key, i, got[i], want[i])
		}
	}
}

func assertNestedString(t *testing.T, m map[string]interface{}, want string, keys ...string) {
	t.Helper()
	cur := m
	for i, k := range keys {
		if i == len(keys)-1 {
			val, _ := cur[k].(string)
			if val != want {
				t.Errorf("path %v: got %q, want %q", keys, val, want)
			}
			return
		}
		next, ok := cur[k].(map[string]interface{})
		if !ok {
			t.Errorf("path %v: key %q not a map", keys, k)
			return
		}
		cur = next
	}
}

func assertSubjectSA(t *testing.T, m map[string]interface{}, saName, namespace string) {
	t.Helper()
	subjectsRaw, ok := m["subjects"]
	if !ok {
		t.Error("missing subjects")
		return
	}
	subjects, ok := subjectsRaw.([]interface{})
	if !ok || len(subjects) == 0 {
		t.Error("subjects empty or wrong type")
		return
	}
	subj, ok := subjects[0].(map[string]interface{})
	if !ok {
		t.Error("subject[0] not a map")
		return
	}
	if subj["kind"] != "ServiceAccount" {
		t.Errorf("subject kind = %q, want ServiceAccount", subj["kind"])
	}
	if subj["name"] != saName {
		t.Errorf("subject name = %q, want %q", subj["name"], saName)
	}
	if subj["namespace"] != namespace {
		t.Errorf("subject namespace = %q, want %q", subj["namespace"], namespace)
	}
}
