package exporter

type ExportOptions struct {
	Namespace string
	Redact    bool
}

type LegacyConfig struct {
	Environment string            `yaml:"environment"`
	Values      map[string]string `yaml:"values,omitempty"`
}

type ExportSummary struct {
	Warnings []string `yaml:"warnings,omitempty"`
	Redacted []string `yaml:"redacted,omitempty"`
}

type LegacyExporter interface {
	Export(envName string, opts ExportOptions) (*LegacyConfig, *ExportSummary, error)
}

