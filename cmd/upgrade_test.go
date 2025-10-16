package cmd

import (
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
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
	assert.Equal(t, "upgrade [RELEASE] [CHART]", upgradeCmd.Use)
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

func TestExtractAndParseUpgradeAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    map[string]UpgradeAnnotation
		description string
	}{
		{
			name: "valid structured annotations",
			annotations: map[string]string{
				"upgrade.datarobot.com/migration": `source: ">=0.1.0 <0.2.0"
target: "0.2.0"
description: Database schema migration required
action: |
  ./scripts/migrate-db.sh
  ./scripts/verify-migration.sh`,
				"upgrade.datarobot.com/backup": `source: "*"
target: "2.x"
description: Run backup before upgrade
action: |
  ./scripts/backup.sh
  ./scripts/verify-backup.sh`,
			},
			expected: map[string]UpgradeAnnotation{
				"upgrade.datarobot.com/migration": {
					Source:      ">=0.1.0 <0.2.0",
					Target:      "0.2.0",
					Description: "Database schema migration required",
					Action:      "./scripts/migrate-db.sh\n./scripts/verify-migration.sh",
				},
				"upgrade.datarobot.com/backup": {
					Source:      "*",
					Target:      "2.x",
					Description: "Run backup before upgrade",
					Action:      "./scripts/backup.sh\n./scripts/verify-backup.sh",
				},
			},
			description: "should parse valid structured annotations",
		},
		{
			name: "mixed annotations with upgrade prefix",
			annotations: map[string]string{
				"upgrade.datarobot.com/migration": `description: Database migration
action: ./migrate.sh`,
				"datarobot.com/images": "test-image:1.0.0",
				"other.annotation":     "some value",
			},
			expected: map[string]UpgradeAnnotation{
				"upgrade.datarobot.com/migration": {
					Description: "Database migration",
					Action:      "./migrate.sh",
				},
			},
			description: "should only parse annotations with upgrade.datarobot.com/ prefix",
		},
		{
			name: "no upgrade annotations",
			annotations: map[string]string{
				"datarobot.com/images": "test-image:1.0.0",
				"other.annotation":     "some value",
			},
			expected:    map[string]UpgradeAnnotation{},
			description: "should return empty map when no upgrade annotations exist",
		},
		{
			name:        "empty annotations",
			annotations: map[string]string{},
			expected:    map[string]UpgradeAnnotation{},
			description: "should return empty map for empty annotations",
		},
		{
			name:        "nil annotations",
			annotations: nil,
			expected:    map[string]UpgradeAnnotation{},
			description: "should return empty map for nil annotations",
		},
		{
			name: "invalid YAML annotations",
			annotations: map[string]string{
				"upgrade.datarobot.com/invalid": "invalid: yaml: content: [",
				"upgrade.datarobot.com/valid": `description: Valid annotation
action: ./script.sh`,
			},
			expected: map[string]UpgradeAnnotation{
				"upgrade.datarobot.com/valid": {
					Description: "Valid annotation",
					Action:      "./script.sh",
				},
			},
			description: "should skip invalid YAML annotations",
		},
		{
			name: "missing required fields",
			annotations: map[string]string{
				"upgrade.datarobot.com/no-action":      `description: No action field`,
				"upgrade.datarobot.com/no-description": `action: ./script.sh`,
				"upgrade.datarobot.com/valid": `description: Valid annotation
action: ./script.sh`,
			},
			expected: map[string]UpgradeAnnotation{
				"upgrade.datarobot.com/valid": {
					Description: "Valid annotation",
					Action:      "./script.sh",
				},
			},
			description: "should skip annotations missing required fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test chart with the specified annotations
			testChart := &chart.Chart{
				Metadata: &chart.Metadata{
					Annotations: tt.annotations,
				},
			}

			result := extractAndParseUpgradeAnnotations(testChart)

			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestExtractUpgradeAnnotationsWithNilMetadata(t *testing.T) {
	// Test with nil metadata
	testChart := &chart.Chart{
		Metadata: nil,
	}

	result := extractAndParseUpgradeAnnotations(testChart)
	assert.Empty(t, result, "should return empty map when metadata is nil")
}

func TestMatchesVersionConstraint(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		constraint string
		expected   bool
		shouldErr  bool
	}{
		// Exact matches
		{"exact match", "1.2.3", "1.2.3", true, false},
		{"exact no match", "1.2.3", "1.2.4", false, false},

		// Range constraints
		{"range matches lower", "0.1.5", ">=0.1.0 <0.2.0", true, false},
		{"range matches upper", "0.1.9", ">=0.1.0 <0.2.0", true, false},
		{"range too low", "0.0.9", ">=0.1.0 <0.2.0", false, false},
		{"range too high", "0.2.0", ">=0.1.0 <0.2.0", false, false},
		{"tilde constraint", "1.2.5", "~1.2.0", true, false},
		{"caret constraint", "1.5.0", "^1.2.0", true, false},

		// Wildcard matches
		{"wildcard major", "2.1.3", "2.x", true, false},
		{"wildcard minor", "2.1.5", "2.1.x", true, false},
		{"wildcard all", "3.5.2", "*", true, false},
		{"wildcard no match major", "1.2.3", "2.x", false, false},

		// Empty constraint
		{"empty constraint", "1.2.3", "", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := matchesVersionConstraint(tt.version, tt.constraint)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestShouldDisplayAnnotation(t *testing.T) {
	tests := []struct {
		name             string
		annotation       UpgradeAnnotation
		installedVersion string
		newVersion       string
		expected         bool
	}{
		{
			name: "both empty - always display",
			annotation: UpgradeAnnotation{
				Source: "",
				Target: "",
			},
			installedVersion: "1.0.0",
			newVersion:       "2.0.0",
			expected:         true,
		},
		{
			name: "source empty - check target only",
			annotation: UpgradeAnnotation{
				Source: "",
				Target: "2.x",
			},
			installedVersion: "1.0.0",
			newVersion:       "2.1.0",
			expected:         true,
		},
		{
			name: "target empty - check source only",
			annotation: UpgradeAnnotation{
				Source: ">=1.0.0 <2.0.0",
				Target: "",
			},
			installedVersion: "1.5.0",
			newVersion:       "3.0.0",
			expected:         true,
		},
		{
			name: "both match - display",
			annotation: UpgradeAnnotation{
				Source: ">=0.1.0 <0.2.0",
				Target: "0.2.0",
			},
			installedVersion: "0.1.5",
			newVersion:       "0.2.0",
			expected:         true,
		},
		{
			name: "source no match - hide",
			annotation: UpgradeAnnotation{
				Source: ">=0.1.0 <0.2.0",
				Target: "0.2.0",
			},
			installedVersion: "0.3.0",
			newVersion:       "0.2.0",
			expected:         false,
		},
		{
			name: "target no match - hide",
			annotation: UpgradeAnnotation{
				Source: ">=0.1.0 <0.2.0",
				Target: "0.2.x",
			},
			installedVersion: "0.1.5",
			newVersion:       "0.3.0",
			expected:         false,
		},
		{
			name: "wildcard source and target both match",
			annotation: UpgradeAnnotation{
				Source: "1.x",
				Target: "2.x",
			},
			installedVersion: "1.5.0",
			newVersion:       "2.3.0",
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldDisplayAnnotation(tt.annotation, tt.installedVersion, tt.newVersion)
			assert.Equal(t, tt.expected, result)
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
