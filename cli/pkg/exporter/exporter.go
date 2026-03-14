package exporter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var secretPattern = regexp.MustCompile(`(?i)(PASSWORD|SECRET|KEY|TOKEN)`)

type ReadOnlyLegacyClient interface {
	GetComposeConfig(envName string, opts ExportOptions) (*LegacyConfig, error)
	InspectRuntime(envName string, opts ExportOptions) (*LegacyConfig, error)
	ListInaccessibleServices(envName string, opts ExportOptions) ([]string, error)
}

type legacyExporter struct {
	client ReadOnlyLegacyClient
}

func NewLegacyExporter(client ReadOnlyLegacyClient) LegacyExporter {
	return &legacyExporter{client: client}
}

func (e *legacyExporter) Export(envName string, opts ExportOptions) (*LegacyConfig, *ExportSummary, error) {
	if e.client == nil {
		return nil, nil, fmt.Errorf("legacy exporter requires a client")
	}

	config, inferred, err := e.loadConfig(envName, opts)
	if err != nil {
		return nil, nil, err
	}

	inaccessible, err := e.client.ListInaccessibleServices(envName, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("list inaccessible services: %w", err)
	}

	summary := &ExportSummary{
		Inaccessible: inaccessible,
	}

	if inferred {
		summary.Warnings = append(summary.Warnings, "configuration inferred from runtime metadata")
	}

	if len(inaccessible) > 0 {
		summary.PartialResult = true
		summary.Warnings = append(summary.Warnings, "partial export completed with inaccessible services")
	}

	if opts.Redact {
		redactConfig(config, summary)
	}

	for i := range config.Services {
		if config.Services[i].Inferred {
			summary.Inferred = append(summary.Inferred, config.Services[i].Name)
		}
	}

	summary.ServiceCount = len(config.Services)
	sort.Strings(summary.Redacted)
	sort.Strings(summary.Inaccessible)
	sort.Strings(summary.Inferred)

	return config, summary, nil
}

func (e *legacyExporter) loadConfig(envName string, opts ExportOptions) (*LegacyConfig, bool, error) {
	config, err := e.client.GetComposeConfig(envName, opts)
	if err == nil {
		return config, false, nil
	}

	runtimeConfig, runtimeErr := e.client.InspectRuntime(envName, opts)
	if runtimeErr != nil {
		return nil, false, fmt.Errorf("cannot access legacy environment %q. no data extracted.", envName)
	}

	for i := range runtimeConfig.Services {
		runtimeConfig.Services[i].Inferred = true
	}

	return runtimeConfig, true, nil
}

func redactConfig(config *LegacyConfig, summary *ExportSummary) {
	for i := range config.Services {
		for key, value := range config.Services[i].Environment {
			if !secretPattern.MatchString(key) {
				continue
			}
			if value != "<REDACTED>" {
				config.Services[i].Environment[key] = "<REDACTED>"
			}
			summary.Redacted = append(summary.Redacted, fmt.Sprintf("%s:%s", config.Services[i].Name, key))
		}
	}
}

func IsSecretKey(key string) bool {
	return secretPattern.MatchString(strings.TrimSpace(key))
}
