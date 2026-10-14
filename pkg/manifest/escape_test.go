package manifest

import (
	"strings"
	"testing"
)

func TestEscapeBraces_BothDelimiters(t *testing.T) {
	in := "description: uses {{ .foo }} and }} alone"
	out := EscapeBraces(in)
	if strings.Contains(out, "{{ .foo }}") {
		t.Fatalf("raw template braces survived unescaped: %q", out)
	}
	if !strings.Contains(out, `{{ "{{" }}`) {
		t.Fatalf("opening brace not escaped: %q", out)
	}
	if !strings.Contains(out, `{{ "}}" }}`) {
		t.Fatalf("closing brace not escaped: %q", out)
	}
}

func TestEscapeBraces_NoBraces(t *testing.T) {
	in := "plain: value"
	if got := EscapeBraces(in); got != in {
		t.Fatalf("unexpected change: %q", got)
	}
}
