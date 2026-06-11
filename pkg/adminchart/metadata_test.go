package adminchart

import (
	"testing"
)

func TestReadSourceChartMeta(t *testing.T) {
	meta, err := ReadSourceChartMeta("../../tests/charts/test-chart6")
	if err != nil {
		t.Fatalf("ReadSourceChartMeta error: %v", err)
	}
	if meta.Name != "test-chart6" {
		t.Errorf("name = %q, want test-chart6", meta.Name)
	}
	if meta.Version != "0.1.0" {
		t.Errorf("version = %q, want 0.1.0", meta.Version)
	}
	if meta.AppVersion != "1.36.1" {
		t.Errorf("appVersion = %q, want 1.36.1", meta.AppVersion)
	}
}

func TestReadSourceChartMeta_InvalidPath(t *testing.T) {
	_, err := ReadSourceChartMeta("/nonexistent/path/chart")
	if err == nil {
		t.Fatal("expected error on invalid path, got nil")
	}
}
