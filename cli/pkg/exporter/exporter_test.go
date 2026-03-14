package exporter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockLegacyClient struct {
	composeConfig       *LegacyConfig
	composeErr          error
	runtimeConfig       *LegacyConfig
	runtimeErr          error
	inaccessible        []string
	inaccessibleErr     error
	writeCallsObserved  int
	composeCalls        int
	runtimeCalls        int
	inaccessibleCalls   int
}

func (m *mockLegacyClient) GetComposeConfig(envName string, opts ExportOptions) (*LegacyConfig, error) {
	m.composeCalls++
	return cloneConfig(m.composeConfig), m.composeErr
}

func (m *mockLegacyClient) InspectRuntime(envName string, opts ExportOptions) (*LegacyConfig, error) {
	m.runtimeCalls++
	return cloneConfig(m.runtimeConfig), m.runtimeErr
}

func (m *mockLegacyClient) ListInaccessibleServices(envName string, opts ExportOptions) ([]string, error) {
	m.inaccessibleCalls++
	return append([]string(nil), m.inaccessible...), m.inaccessibleErr
}

func TestExportRedactsSensitiveVariables(t *testing.T) {
	t.Parallel()

	client := &mockLegacyClient{
		composeConfig: &LegacyConfig{
			Environment: "legacy-a",
			Services: []LegacyService{{
				Name:  "api",
				Image: "ghcr.io/example/api:v1",
				Environment: map[string]string{
					"DB_PASSWORD":    "secret",
					"API_SECRET_KEY": "secret-key",
					"OAUTH_TOKEN":    "token",
					"AWS_ACCESS_KEY": "access",
					"LOG_LEVEL":      "debug",
				},
			}},
		},
	}

	exporter := NewLegacyExporter(client)
	config, summary, err := exporter.Export("legacy-a", ExportOptions{Redact: true})
	require.NoError(t, err)

	env := config.Services[0].Environment
	require.Equal(t, "<REDACTED>", env["DB_PASSWORD"])
	require.Equal(t, "<REDACTED>", env["API_SECRET_KEY"])
	require.Equal(t, "<REDACTED>", env["OAUTH_TOKEN"])
	require.Equal(t, "<REDACTED>", env["AWS_ACCESS_KEY"])
	require.Equal(t, "debug", env["LOG_LEVEL"])
	require.Len(t, summary.Redacted, 4)
	require.Zero(t, client.writeCallsObserved)
}

func TestExportReturnsPartialResultForInaccessibleServices(t *testing.T) {
	t.Parallel()

	client := &mockLegacyClient{
		composeConfig: &LegacyConfig{
			Environment: "legacy-b",
			Services: []LegacyService{{
				Name:  "frontend",
				Image: "nginx:1.27",
			}},
		},
		inaccessible: []string{"worker", "payments"},
	}

	exporter := NewLegacyExporter(client)
	config, summary, err := exporter.Export("legacy-b", ExportOptions{Redact: true})
	require.NoError(t, err)
	require.Len(t, config.Services, 1)
	require.ElementsMatch(t, []string{"worker", "payments"}, summary.Inaccessible)
	require.True(t, summary.PartialResult)
}

func TestExportFallsBackToRuntimeInspection(t *testing.T) {
	t.Parallel()

	client := &mockLegacyClient{
		composeErr: errors.New("compose unavailable"),
		runtimeConfig: &LegacyConfig{
			Environment: "legacy-c",
			Services: []LegacyService{{
				Name:  "api",
				Image: "ghcr.io/example/api:v2",
			}},
		},
	}

	exporter := NewLegacyExporter(client)
	config, summary, err := exporter.Export("legacy-c", ExportOptions{Redact: true})
	require.NoError(t, err)
	require.Len(t, config.Services, 1)
	require.True(t, config.Services[0].Inferred)
	require.Contains(t, summary.Inferred, "api")
	require.Equal(t, 1, client.composeCalls)
	require.Equal(t, 1, client.runtimeCalls)
	require.Zero(t, client.writeCallsObserved)
}

func TestExportFailsWhenNoAccessExists(t *testing.T) {
	t.Parallel()

	client := &mockLegacyClient{
		composeErr: errors.New("compose unavailable"),
		runtimeErr: errors.New("runtime unavailable"),
	}

	exporter := NewLegacyExporter(client)
	_, _, err := exporter.Export("legacy-d", ExportOptions{Redact: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), `cannot access legacy environment "legacy-d". no data extracted.`)
}

func TestIsSecretKey(t *testing.T) {
	t.Parallel()

	require.True(t, IsSecretKey("DB_PASSWORD"))
	require.True(t, IsSecretKey("API_SECRET_KEY"))
	require.False(t, IsSecretKey("LOG_LEVEL"))
}

func cloneConfig(cfg *LegacyConfig) *LegacyConfig {
	if cfg == nil {
		return nil
	}

	out := &LegacyConfig{
		Environment: cfg.Environment,
		Namespace:   cfg.Namespace,
		Services:    make([]LegacyService, 0, len(cfg.Services)),
	}

	for _, service := range cfg.Services {
		cloned := service
		if service.Environment != nil {
			cloned.Environment = map[string]string{}
			for key, value := range service.Environment {
				cloned.Environment[key] = value
			}
		}
		if service.Ports != nil {
			cloned.Ports = append([]string(nil), service.Ports...)
		}
		if service.Volumes != nil {
			cloned.Volumes = append([]string(nil), service.Volumes...)
		}
		if service.Dependencies != nil {
			cloned.Dependencies = append([]string(nil), service.Dependencies...)
		}
		out.Services = append(out.Services, cloned)
	}

	return out
}
