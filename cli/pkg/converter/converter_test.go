package converter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fixture  string
		assertFn func(t *testing.T, manifest *EnvironmentManifest, report *ConversionReport)
	}{
		{
			name:    "simple-compose",
			fixture: "../../testdata/fixtures/simple-compose.yaml",
			assertFn: func(t *testing.T, manifest *EnvironmentManifest, report *ConversionReport) {
				require.Len(t, manifest.Spec.Components, 1)
				require.Equal(t, "webapp", manifest.Spec.Components[0].Type)
				require.True(t, manifest.Spec.Components[0].Enabled)
				require.Contains(t, report.MappedServices, "frontend")
			},
		},
		{
			name:    "with-database",
			fixture: "../../testdata/fixtures/with-database.yaml",
			assertFn: func(t *testing.T, manifest *EnvironmentManifest, report *ConversionReport) {
				require.Len(t, manifest.Spec.Components, 2)
				require.Equal(t, []string{"api", "db"}, []string{manifest.Spec.Components[0].Name, manifest.Spec.Components[1].Name})
				require.Equal(t, []string{"db"}, report.ServiceDependsOn["api"])
			},
		},
		{
			name:    "unsupported-features",
			fixture: "../../testdata/fixtures/unsupported-features.yaml",
			assertFn: func(t *testing.T, manifest *EnvironmentManifest, report *ConversionReport) {
				require.Len(t, manifest.Spec.Components, 1)
				require.NotEmpty(t, report.ManualReview)
				require.Condition(t, func() bool {
					for _, warning := range report.Warnings {
						if strings.Contains(warning, "unsupported build directive") {
							return true
						}
					}
					return false
				})
			},
		},
		{
			name:    "unknown-service-type",
			fixture: "../../testdata/fixtures/unknown-service-type.yaml",
			assertFn: func(t *testing.T, manifest *EnvironmentManifest, report *ConversionReport) {
				require.Len(t, manifest.Spec.Components, 1)
				require.False(t, manifest.Spec.Components[0].Enabled)
				require.Contains(t, report.DisabledServices, "mystery")
			},
		},
		{
			name:    "empty-compose",
			fixture: "../../testdata/fixtures/empty-compose.yaml",
			assertFn: func(t *testing.T, manifest *EnvironmentManifest, report *ConversionReport) {
				require.Empty(t, manifest.Spec.Components)
				require.Contains(t, report.Warnings, "compose file contains no services")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := mustReadFile(t, tt.fixture)
			converter := NewComposeConverter()

			manifest, report, err := converter.Convert(bytes.NewReader(data))
			require.NoError(t, err)
			require.NotNil(t, manifest)
			require.NotNil(t, report)
			tt.assertFn(t, manifest, report)
		})
	}
}

func TestConvertDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	input := mustReadFile(t, "../../testdata/fixtures/with-database.yaml")
	original := append([]byte(nil), input...)

	converter := NewComposeConverter()
	_, _, err := converter.Convert(bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, original, input)
}

func TestSimpleComposeMatchesExpectedFixture(t *testing.T) {
	t.Parallel()

	data := mustReadFile(t, "../../testdata/fixtures/simple-compose.yaml")
	expected := mustReadFile(t, "../../testdata/expected/simple-compose-expected.yaml")

	converter := NewComposeConverter()
	manifest, _, err := converter.Convert(bytes.NewReader(data))
	require.NoError(t, err)

	rendered, err := MarshalYAML(manifest)
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(string(rendered)))
}

func mustReadFile(t *testing.T, rel string) []byte {
	t.Helper()

	path := filepath.Clean(rel)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
