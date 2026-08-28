package crdchart

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadSourceChartMeta(t *testing.T) {
	// test-chart6 exists in the repo fixtures.
	meta, err := ReadSourceChartMeta("../../tests/charts/test-chart6/")
	assert.NoError(t, err)
	assert.Equal(t, "test-chart6", meta.Name)
	assert.NotEmpty(t, meta.Version)
}

func TestReadSourceChartMetaMissing(t *testing.T) {
	_, err := ReadSourceChartMeta("../../tests/charts/does-not-exist/")
	assert.Error(t, err)
}
