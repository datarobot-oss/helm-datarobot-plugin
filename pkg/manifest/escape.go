package manifest

import "strings"

// EscapeBraces neutralizes Go-template delimiters that appear inside resource
// bodies (CEL rules, descriptions) so Helm renders them back literally instead
// of trying to evaluate them. Both "{{" and "}}" are escaped.
func EscapeBraces(raw string) string {
	const lo = "\x00LBRACE\x00"
	const ro = "\x00RBRACE\x00"
	s := strings.ReplaceAll(raw, "{{", lo)
	s = strings.ReplaceAll(s, "}}", ro)
	s = strings.ReplaceAll(s, lo, `{{ "{{" }}`)
	s = strings.ReplaceAll(s, ro, `{{ "}}" }}`)
	return s
}
