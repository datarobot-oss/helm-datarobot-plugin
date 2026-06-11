package manifest

import (
	"encoding/json"
	"fmt"
	"strings"

	sigsyaml "sigs.k8s.io/yaml"
)

// roleDoc is used to unmarshal only the fields we need from a Role resource.
type roleDoc struct {
	Rules []map[string]interface{} `json:"rules"`
}

// RoleRules returns the union of .rules entries from all kind: Role resources.
// Roles with missing or empty rules are skipped. Exact-duplicate rules are
// deduplicated by canonical JSON serialization. Order: first-seen.
//
// Only resources with apiVersion prefixed by "rbac.authorization.k8s.io/" are
// matched. CRD-defined kinds whose name happens to be "Role" (e.g.
// roles.rabbitmq.com with apiVersion rabbitmq.com/v1) are ignored.
//
// Deduplication is key-order-insensitive (via canonical JSON) but
// element-order-sensitive within verbs/resources/apiGroups slices; rules that
// are logically identical but differ in element order are kept as duplicates
// (harmless — the union is still correct; just not minimal).
func RoleRules(resources []Resource) ([]map[string]interface{}, error) {
	seen := make(map[string]struct{})
	var union []map[string]interface{}

	for _, r := range resources {
		if r.Kind != "Role" {
			continue
		}
		if !strings.HasPrefix(r.APIVersion, "rbac.authorization.k8s.io/") {
			continue
		}

		var doc roleDoc
		if err := sigsyaml.Unmarshal([]byte(r.RawYAML), &doc); err != nil {
			return nil, fmt.Errorf("RoleRules: role %q: failed to unmarshal: %w", r.Name, err)
		}

		if len(doc.Rules) == 0 {
			continue
		}

		for _, rule := range doc.Rules {
			key, err := json.Marshal(rule)
			if err != nil {
				return nil, fmt.Errorf("RoleRules: role %q: failed to serialize rule: %w", r.Name, err)
			}
			ks := string(key)
			if _, dup := seen[ks]; dup {
				continue
			}
			seen[ks] = struct{}{}
			union = append(union, rule)
		}
	}

	return union, nil
}
