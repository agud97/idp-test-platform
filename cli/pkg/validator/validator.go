package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/agud97/idp-platform/cli/pkg/converter"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apivalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

var imageTagPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$`)
var configKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var sensitiveConfigKeyPattern = regexp.MustCompile(`(?i)(PASSWORD|PASS|SECRET|TOKEN|PRIVATE_?KEY|ACCESS_?KEY|API_?KEY)`)

type environmentValidator struct {
	schemaValidator apivalidation.SchemaValidator
}

func NewEnvironmentValidator() (EnvironmentValidator, error) {
	crdPath, err := environmentCRDPath()
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(crdPath)
	if err != nil {
		return nil, fmt.Errorf("read environment crd: %w", err)
	}

	var crdV1 apiextensionsv1.CustomResourceDefinition
	if err := yaml.Unmarshal(raw, &crdV1); err != nil {
		return nil, fmt.Errorf("unmarshal environment crd: %w", err)
	}

	schemaV1, err := firstServedSchema(&crdV1)
	if err != nil {
		return nil, err
	}

	var schema apiextensions.JSONSchemaProps
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(schemaV1, &schema, nil); err != nil {
		return nil, fmt.Errorf("convert environment schema: %w", err)
	}

	validator, _, err := apivalidation.NewSchemaValidator(&schema)
	if err != nil {
		return nil, fmt.Errorf("create schema validator: %w", err)
	}

	return &environmentValidator{schemaValidator: validator}, nil
}

func environmentCRDPath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("locate validator source: runtime caller unavailable")
	}

	candidates := []string{
		filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "platform", "crds", "environment.yaml"),
		filepath.Join(filepath.Dir(currentFile), "..", "..", "platform", "crds", "environment.yaml"),
		filepath.Join(".", "platform", "crds", "environment.yaml"),
	}

	for _, candidate := range candidates {
		crdPath := filepath.Clean(candidate)
		if _, err := os.Stat(crdPath); err == nil {
			return crdPath, nil
		}
	}

	return "", fmt.Errorf("read environment crd: %w", os.ErrNotExist)
}

func firstServedSchema(crd *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.JSONSchemaProps, error) {
	for _, version := range crd.Spec.Versions {
		if !version.Served || version.Schema == nil || version.Schema.OpenAPIV3Schema == nil {
			continue
		}
		return version.Schema.OpenAPIV3Schema, nil
	}

	return nil, fmt.Errorf("environment crd does not define a served openAPIV3 schema")
}

func (v *environmentValidator) Validate(manifest *converter.EnvironmentManifest) []ValidationError {
	if manifest == nil {
		return []ValidationError{{Path: "$", Message: "manifest is required"}}
	}

	errors := make([]ValidationError, 0)
	errors = append(errors, v.validateAgainstCRDSchema(manifest)...)
	errors = append(errors, validateDuplicateNames(manifest)...)
	errors = append(errors, validateReplicas(manifest)...)
	errors = append(errors, validateImageTags(manifest)...)
	errors = append(errors, validateConfigOverrideKeys(manifest)...)
	return dedupe(errors)
}

func (v *environmentValidator) validateAgainstCRDSchema(manifest *converter.EnvironmentManifest) []ValidationError {
	payload, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(manifest)
	if err != nil {
		return []ValidationError{{Path: "$", Message: fmt.Sprintf("cannot convert manifest: %v", err)}}
	}
	if payload["status"] == nil {
		delete(payload, "status")
	}

	errs := apivalidation.ValidateCustomResource(nil, payload, v.schemaValidator)
	out := make([]ValidationError, 0, len(errs))
	for _, err := range errs {
		out = append(out, ValidationError{
			Path:    err.Field,
			Message: err.ErrorBody(),
		})
	}
	return out
}

func validateDuplicateNames(manifest *converter.EnvironmentManifest) []ValidationError {
	seen := map[string]int{}
	var out []ValidationError
	for i, component := range manifest.Spec.Components {
		if other, ok := seen[component.Name]; ok {
			out = append(out, ValidationError{
				Path:    fmt.Sprintf("spec.components[%d].name", i),
				Message: fmt.Sprintf("duplicate component name %q also used at spec.components[%d].name", component.Name, other),
			})
			continue
		}
		seen[component.Name] = i
	}
	return out
}

func validateReplicas(manifest *converter.EnvironmentManifest) []ValidationError {
	var out []ValidationError
	for i, component := range manifest.Spec.Components {
		if component.Replicas == nil {
			continue
		}
		if *component.Replicas < 0 || *component.Replicas > 50 {
			out = append(out, ValidationError{
				Path:    fmt.Sprintf("spec.components[%d].replicas", i),
				Message: "replicas must be in the allowed range 0–50",
			})
		}
	}
	return out
}

func validateImageTags(manifest *converter.EnvironmentManifest) []ValidationError {
	var out []ValidationError
	for i, component := range manifest.Spec.Components {
		if component.ImageTag == "" {
			continue
		}
		if !imageTagPattern.MatchString(component.ImageTag) {
			out = append(out, ValidationError{
				Path:    fmt.Sprintf("spec.components[%d].imageTag", i),
				Message: "imageTag must be a valid OCI tag",
			})
		}
	}
	return out
}

func validateConfigOverrideKeys(manifest *converter.EnvironmentManifest) []ValidationError {
	var out []ValidationError
	for i, component := range manifest.Spec.Components {
		if len(component.ConfigOverrides) == 0 {
			continue
		}
		for key := range component.ConfigOverrides {
			switch {
			case !configKeyPattern.MatchString(key):
				out = append(out, ValidationError{
					Path:    fmt.Sprintf("spec.components[%d].configOverrides.%s", i, key),
					Message: fmt.Sprintf("invalid key %q for component type %q", key, component.Type),
				})
			case sensitiveConfigKeyPattern.MatchString(key):
				out = append(out, ValidationError{
					Path:    fmt.Sprintf("spec.components[%d].configOverrides.%s", i, key),
					Message: fmt.Sprintf("sensitive key %q is not allowed in configOverrides for component type %q", key, component.Type),
				})
			}
		}
	}
	return out
}

func dedupe(in []ValidationError) []ValidationError {
	seen := map[string]struct{}{}
	out := make([]ValidationError, 0, len(in))
	for _, err := range in {
		keyBytes, _ := json.Marshal(err)
		key := string(keyBytes)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		err.Message = strings.TrimSpace(err.Message)
		out = append(out, err)
	}
	return out
}
