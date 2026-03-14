package converter

import "io"

type EnvironmentManifest struct {
	APIVersion string             `yaml:"apiVersion"`
	Kind       string             `yaml:"kind"`
	Metadata   Metadata           `yaml:"metadata"`
	Spec       EnvironmentSpec    `yaml:"spec"`
	Status     *EnvironmentStatus `yaml:"status,omitempty"`
}

type Metadata struct {
	Name      string            `yaml:"name,omitempty"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

type EnvironmentSpec struct {
	Owner      string      `yaml:"owner"`
	Team       string      `yaml:"team"`
	Components []Component `yaml:"components"`
}

type Component struct {
	Name            string            `yaml:"name"`
	Type            string            `yaml:"type"`
	Enabled         bool              `yaml:"enabled"`
	ImageTag        string            `yaml:"imageTag,omitempty"`
	Replicas        *int32            `yaml:"replicas,omitempty"`
	ConfigOverrides map[string]string `yaml:"configOverrides,omitempty"`
}

type EnvironmentStatus struct {
	Phase            string `yaml:"phase,omitempty"`
	LastSyncedCommit string `yaml:"lastSyncedCommit,omitempty"`
	Message          string `yaml:"message,omitempty"`
}

type ConversionReport struct {
	Warnings          []string            `yaml:"warnings,omitempty"`
	Errors            []string            `yaml:"errors,omitempty"`
	MappedServices    []string            `yaml:"mappedServices,omitempty"`
	DisabledServices  []string            `yaml:"disabledServices,omitempty"`
	ManualReview      []string            `yaml:"manualReview,omitempty"`
	ServicePorts      map[string][]int32  `yaml:"servicePorts,omitempty"`
	ServiceVolumes    map[string][]string `yaml:"serviceVolumes,omitempty"`
	ServiceDependsOn  map[string][]string `yaml:"serviceDependsOn,omitempty"`
	DetectedNamespace string              `yaml:"detectedNamespace,omitempty"`
}

type ComposeConverter interface {
	Convert(input io.Reader) (*EnvironmentManifest, *ConversionReport, error)
}
