package exporter

type ExportOptions struct {
	Namespace string
	Redact    bool
}

type LegacyService struct {
	Name         string            `yaml:"name"`
	Image        string            `yaml:"image,omitempty"`
	Ports        []string          `yaml:"ports,omitempty"`
	Environment  map[string]string `yaml:"environment,omitempty"`
	Volumes      []string          `yaml:"volumes,omitempty"`
	Dependencies []string          `yaml:"dependencies,omitempty"`
	Inferred     bool              `yaml:"inferred,omitempty"`
}

type LegacyConfig struct {
	Environment string          `yaml:"environment"`
	Namespace   string          `yaml:"namespace,omitempty"`
	Services    []LegacyService `yaml:"services,omitempty"`
}

type ExportSummary struct {
	Warnings      []string `yaml:"warnings,omitempty"`
	Redacted      []string `yaml:"redacted,omitempty"`
	Inaccessible  []string `yaml:"inaccessible,omitempty"`
	Inferred      []string `yaml:"inferred,omitempty"`
	ServiceCount  int      `yaml:"serviceCount,omitempty"`
	PartialResult bool     `yaml:"partialResult,omitempty"`
}

type LegacyExporter interface {
	Export(envName string, opts ExportOptions) (*LegacyConfig, *ExportSummary, error)
}
