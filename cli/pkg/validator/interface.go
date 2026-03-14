package validator

import "github.com/agud97/idp-platform/cli/pkg/converter"

type ValidationError struct {
	Path    string `yaml:"path"`
	Message string `yaml:"message"`
}

type EnvironmentValidator interface {
	Validate(manifest *converter.EnvironmentManifest) []ValidationError
}
