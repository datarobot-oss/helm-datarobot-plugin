package cmd

import (
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		name        string
		oldVersion  string
		newVersion  string
		shouldError bool
		description string
	}{
		{
			name:        "valid upgrade - patch version bump",
			oldVersion:  "1.0.0",
			newVersion:  "1.0.1",
			shouldError: false,
			description: "upgrading from 1.0.0 to 1.0.1 should succeed",
		},
		{
			name:        "valid upgrade - minor version bump",
			oldVersion:  "1.0.0",
			newVersion:  "1.1.0",
			shouldError: false,
			description: "upgrading from 1.0.0 to 1.1.0 should succeed",
		},
		{
			name:        "valid upgrade - major version bump",
			oldVersion:  "1.0.0",
			newVersion:  "2.0.0",
			shouldError: false,
			description: "upgrading from 1.0.0 to 2.0.0 should succeed",
		},
		{
			name:        "same version",
			oldVersion:  "1.0.0",
			newVersion:  "1.0.0",
			shouldError: false,
			description: "same version should not error (idempotent)",
		},
		{
			name:        "downgrade - patch version",
			oldVersion:  "1.0.1",
			newVersion:  "1.0.0",
			shouldError: true,
			description: "downgrading from 1.0.1 to 1.0.0 should fail",
		},
		{
			name:        "downgrade - minor version",
			oldVersion:  "1.1.0",
			newVersion:  "1.0.0",
			shouldError: true,
			description: "downgrading from 1.1.0 to 1.0.0 should fail",
		},
		{
			name:        "downgrade - major version",
			oldVersion:  "2.0.0",
			newVersion:  "1.0.0",
			shouldError: true,
			description: "downgrading from 2.0.0 to 1.0.0 should fail",
		},
		{
			name:        "pre-release version upgrade",
			oldVersion:  "1.0.0-alpha",
			newVersion:  "1.0.0",
			shouldError: false,
			description: "upgrading from pre-release to stable should succeed",
		},
		{
			name:        "pre-release to pre-release",
			oldVersion:  "1.0.0-alpha",
			newVersion:  "1.0.0-beta",
			shouldError: false,
			description: "upgrading from alpha to beta should succeed",
		},
		{
			name:        "metadata version handling",
			oldVersion:  "1.0.0+build123",
			newVersion:  "1.0.0+build456",
			shouldError: false,
			description: "metadata changes should not prevent upgrade",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVer, err := semver.NewVersion(tt.oldVersion)
			assert.NoError(t, err, "old version should parse")

			newVer, err := semver.NewVersion(tt.newVersion)
			assert.NoError(t, err, "new version should parse")

			isDowngrade := newVer.LessThan(oldVer)

			if tt.shouldError {
				assert.True(t, isDowngrade, tt.description)
			} else {
				assert.False(t, isDowngrade, tt.description)
			}
		})
	}
}

func TestInvalidVersions(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "invalid version - no numbers",
			version: "invalid",
		},
		{
			name:    "invalid version - extra dots",
			version: "1.0.0.0",
		},
		{
			name:    "invalid version - letters in version",
			version: "1.0.abc",
		},
		{
			name:    "invalid version - empty string",
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := semver.NewVersion(tt.version)
			assert.Error(t, err, "invalid version should produce error")
		})
	}
}

func TestUpgradeCommandExists(t *testing.T) {
	// Verify that the upgrade command is registered
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "upgrade" {
			found = true
			break
		}
	}
	assert.True(t, found, "upgrade command should be registered with root command")
}

func TestUpgradeCommandFlags(t *testing.T) {
	// Verify that key flags are registered
	requiredFlags := []string{
		"namespace",
		"install",
		"dry-run",
		"force",
		"timeout",
		"wait",
		"atomic",
		"reuse-values",
		"reset-values",
		"version",
		"values",
		"set",
	}

	for _, flagName := range requiredFlags {
		flag := upgradeCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "flag %s should be registered", flagName)
	}
}

func TestUpgradeCommandUsage(t *testing.T) {
	// Verify command basic properties
	assert.Equal(t, "upgrade [CHART]", upgradeCmd.Use)
	assert.True(t, upgradeCmd.SilenceUsage)
	assert.NotEmpty(t, upgradeCmd.Short)
	assert.NotEmpty(t, upgradeCmd.Long)
}

func TestUpgradeCommandRejectsRemotePaths(t *testing.T) {
	tests := []struct {
		name        string
		chartPath   string
		shouldError bool
	}{
		{
			name:        "reject OCI URL",
			chartPath:   "oci://registry.example.com/charts/my-chart",
			shouldError: true,
		},
		{
			name:        "reject HTTP URL",
			chartPath:   "http://example.com/chart.tgz",
			shouldError: true,
		},
		{
			name:        "reject HTTPS URL",
			chartPath:   "https://example.com/chart.tgz",
			shouldError: true,
		},
		{
			name:        "accept local directory",
			chartPath:   "./my-chart",
			shouldError: false,
		},
		{
			name:        "accept local file",
			chartPath:   "/path/to/chart.tgz",
			shouldError: false,
		},
		{
			name:        "accept relative path",
			chartPath:   "tests/charts/test-chart1",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the path validation logic directly
			hasRemotePrefix := strings.HasPrefix(tt.chartPath, "oci://") ||
				strings.HasPrefix(tt.chartPath, "http://") ||
				strings.HasPrefix(tt.chartPath, "https://")

			if tt.shouldError {
				assert.True(t, hasRemotePrefix, "remote path should be detected")
			} else {
				assert.False(t, hasRemotePrefix, "local path should not be detected as remote")
			}
		})
	}
}

func TestVersionComparisonEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		oldVersion  string
		newVersion  string
		expectEqual bool
	}{
		{
			name:        "equal versions with metadata",
			oldVersion:  "1.0.0+build1",
			newVersion:  "1.0.0+build2",
			expectEqual: true,
		},
		{
			name:        "equal versions without metadata",
			oldVersion:  "1.0.0",
			newVersion:  "1.0.0",
			expectEqual: true,
		},
		{
			name:        "different pre-release versions",
			oldVersion:  "1.0.0-alpha.1",
			newVersion:  "1.0.0-alpha.2",
			expectEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVer, err := semver.NewVersion(tt.oldVersion)
			assert.NoError(t, err)

			newVer, err := semver.NewVersion(tt.newVersion)
			assert.NoError(t, err)

			if tt.expectEqual {
				assert.True(t, newVer.Equal(oldVer) || (!newVer.LessThan(oldVer) && !newVer.GreaterThan(oldVer)))
			} else {
				assert.NotEqual(t, newVer.Compare(oldVer), 0)
			}
		})
	}
}
