package converter

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

const (
	defaultOwner = "fixture-user"
	defaultTeam  = "fixture-team"
)

type composeConverter struct{}

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(PASSWORD|PASS|SECRET|TOKEN|PRIVATE_?KEY|ACCESS_?KEY|API_?KEY)`)

func NewComposeConverter() ComposeConverter {
	return &composeConverter{}
}

func (c *composeConverter) Convert(input io.Reader) (*EnvironmentManifest, *ConversionReport, error) {
	raw, err := io.ReadAll(input)
	if err != nil {
		return nil, nil, fmt.Errorf("read compose input: %w", err)
	}

	project, err := loadProject(raw)
	if err != nil {
		return nil, nil, err
	}

	manifest := &EnvironmentManifest{
		APIVersion: "idp.platform.io/v1alpha1",
		Kind:       "Environment",
		Metadata: Metadata{
			Name:      normalizeName(project.Name),
			Namespace: buildNamespace(defaultTeam, normalizeName(project.Name)),
		},
		Spec: EnvironmentSpec{
			Owner:      defaultOwner,
			Team:       defaultTeam,
			Components: make([]Component, 0, len(project.Services)),
		},
	}

	report := &ConversionReport{
		ServicePorts:      map[string][]int32{},
		ServiceVolumes:    map[string][]string{},
		ServiceDependsOn:  map[string][]string{},
		DetectedNamespace: manifest.Metadata.Namespace,
	}

	for _, service := range project.Services {
		component, warnings := mapService(service)
		manifest.Spec.Components = append(manifest.Spec.Components, component)
		if component.Enabled {
			report.MappedServices = append(report.MappedServices, service.Name)
		} else {
			report.DisabledServices = append(report.DisabledServices, service.Name)
		}
		report.Warnings = append(report.Warnings, warnings...)
		for _, warning := range warnings {
			if strings.Contains(warning, "manual review") || strings.Contains(warning, "TODO") {
				report.ManualReview = append(report.ManualReview, warning)
			}
		}

		if ports := extractPorts(service); len(ports) > 0 {
			report.ServicePorts[service.Name] = ports
		}
		if volumes := extractVolumes(service); len(volumes) > 0 {
			report.ServiceVolumes[service.Name] = volumes
		}
		if deps := extractDependsOn(service); len(deps) > 0 {
			report.ServiceDependsOn[service.Name] = deps
		}
	}

	if len(manifest.Spec.Components) == 0 {
		report.Warnings = append(report.Warnings, "compose file contains no services")
	}

	sort.Strings(report.MappedServices)
	sort.Strings(report.DisabledServices)
	sort.Strings(report.ManualReview)

	return manifest, report, nil
}

func loadProject(raw []byte) (*types.Project, error) {
	details := types.ConfigDetails{
		WorkingDir:  ".",
		Environment: map[string]string{},
		ConfigFiles: []types.ConfigFile{{
			Filename: "docker-compose.yaml",
			Content:  raw,
		}},
	}

	project, err := loader.Load(details, func(options *loader.Options) {
		options.SetProjectName("converted", true)
		options.SkipValidation = false
	})
	if err != nil {
		return nil, fmt.Errorf("load compose project: %w", err)
	}

	if project.Name == "" {
		project.Name = "converted"
	}

	return project, nil
}

func mapService(service types.ServiceConfig) (Component, []string) {
	componentType, enabled := detectComponentType(service)
	configOverrides, configWarnings := extractEnvironment(service)
	component := Component{
		Name:            service.Name,
		Type:            componentType,
		Enabled:         enabled,
		ImageTag:        extractImageTag(service.Image),
		ConfigOverrides: configOverrides,
	}

	replicas := int32(1)
	component.Replicas = &replicas

	var warnings []string
	warnings = append(warnings, configWarnings...)

	if !enabled {
		warnings = append(warnings, fmt.Sprintf("service %q disabled: TODO no matching component type found", service.Name))
	}

	if service.Build != nil {
		warnings = append(warnings, fmt.Sprintf("service %q requires manual review: unsupported build directive", service.Name))
	}

	if len(service.Networks) > 0 {
		warnings = append(warnings, fmt.Sprintf("service %q requires manual review: custom networks %s", service.Name, strings.Join(sortedKeys(service.Networks), ", ")))
	}

	if volumes := extractVolumes(service); len(volumes) > 0 {
		warnings = append(warnings, fmt.Sprintf("service %q volume mounts require manual review: %s", service.Name, strings.Join(volumes, ", ")))
	}

	if deps := extractDependsOn(service); len(deps) > 0 {
		warnings = append(warnings, fmt.Sprintf("service %q depends_on preserved for review: %s", service.Name, strings.Join(deps, ", ")))
	}

	return component, warnings
}

func detectComponentType(service types.ServiceConfig) (string, bool) {
	name := strings.ToLower(service.Name)
	image := strings.ToLower(service.Image)

	switch {
	case strings.Contains(name, "postgres"), strings.Contains(name, "db"), strings.Contains(image, "postgres"):
		return "postgresql", true
	case strings.Contains(name, "redis"), strings.Contains(image, "redis"):
		return "redis", true
	case strings.Contains(name, "web"), strings.Contains(name, "api"), strings.Contains(name, "frontend"), strings.Contains(name, "backend"), strings.Contains(image, "nginx"), strings.Contains(image, "httpd"), strings.Contains(image, "ghcr.io/example"), strings.Contains(image, "library/nginx"), strings.Contains(image, "/web"), strings.Contains(image, "/api"):
		return "webapp", true
	default:
		return "unknown", false
	}
}

func extractImageTag(image string) string {
	base := filepath.Base(image)
	if idx := strings.LastIndex(base, ":"); idx >= 0 && idx < len(base)-1 {
		return base[idx+1:]
	}
	return ""
}

func extractEnvironment(service types.ServiceConfig) (map[string]string, []string) {
	if len(service.Environment) == 0 {
		return nil, nil
	}

	values := map[string]string{}
	var warnings []string
	for key, value := range service.Environment {
		actual := ""
		if value == nil {
			actual = ""
		} else {
			actual = *value
		}

		if isSensitiveConfig(key, actual) {
			warnings = append(warnings, fmt.Sprintf("service %q config %q omitted from manifest: manual re-entry required", service.Name, key))
			continue
		}

		values[key] = actual
	}

	if len(values) == 0 {
		return nil, warnings
	}

	return values, warnings
}

func isSensitiveConfig(key, value string) bool {
	if IsSensitiveConfigKey(key) {
		return true
	}

	return HasCredentialURL(value)
}

func IsSensitiveConfigKey(key string) bool {
	return sensitiveKeyPattern.MatchString(strings.TrimSpace(key))
}

func HasCredentialURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.User == nil {
		return false
	}

	username := parsed.User.Username()
	password, hasPassword := parsed.User.Password()
	return username != "" || (hasPassword && password != "")
}

func extractPorts(service types.ServiceConfig) []int32 {
	if len(service.Ports) == 0 {
		return nil
	}

	ports := make([]int32, 0, len(service.Ports))
	for _, port := range service.Ports {
		if port.Target > 0 {
			ports = append(ports, int32(port.Target))
		}
	}
	return ports
}

func extractVolumes(service types.ServiceConfig) []string {
	if len(service.Volumes) == 0 {
		return nil
	}

	volumes := make([]string, 0, len(service.Volumes))
	for _, volume := range service.Volumes {
		switch {
		case volume.Target != "":
			volumes = append(volumes, volume.Target)
		case volume.Source != "":
			volumes = append(volumes, volume.Source)
		}
	}
	return volumes
}

func extractDependsOn(service types.ServiceConfig) []string {
	if len(service.DependsOn) == 0 {
		return nil
	}
	deps := make([]string, 0, len(service.DependsOn))
	for name := range service.DependsOn {
		deps = append(deps, name)
	}
	sort.Strings(deps)
	return deps
}

func normalizeName(name string) string {
	if name == "" {
		return "converted"
	}
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func buildNamespace(team, envName string) string {
	teamPart := truncate(team, 20)
	envPart := envName
	base := fmt.Sprintf("env-%s-%s", teamPart, envPart)
	if len(base) <= 63 {
		return base
	}

	n := 63 - 4 - len(teamPart) - 1 - 1 - 4
	if n < 1 {
		n = 1
	}
	envPart = truncate(envName, n)
	hash := hash4(team + "-" + envName)
	return fmt.Sprintf("env-%s-%s-%s", teamPart, envPart, hash)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func hash4(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:4]
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func MarshalYAML(manifest *EnvironmentManifest) ([]byte, error) {
	var buf bytes.Buffer
	writeLine := func(indent int, line string) {
		buf.WriteString(strings.Repeat(" ", indent))
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	writeLine(0, "apiVersion: "+manifest.APIVersion)
	writeLine(0, "kind: "+manifest.Kind)
	writeLine(0, "metadata:")
	writeLine(2, "name: "+manifest.Metadata.Name)
	if manifest.Metadata.Namespace != "" {
		writeLine(2, "namespace: "+manifest.Metadata.Namespace)
	}
	writeLine(0, "spec:")
	writeLine(2, "owner: "+manifest.Spec.Owner)
	writeLine(2, "team: "+manifest.Spec.Team)
	writeLine(2, "components:")
	for _, component := range manifest.Spec.Components {
		writeLine(4, "- name: "+component.Name)
		writeLine(6, "type: "+component.Type)
		writeLine(6, "enabled: "+strconv.FormatBool(component.Enabled))
		if component.ImageTag != "" {
			writeLine(6, "imageTag: "+strconv.Quote(component.ImageTag))
		}
		if component.Replicas != nil {
			writeLine(6, fmt.Sprintf("replicas: %d", *component.Replicas))
		}
		if len(component.ConfigOverrides) > 0 {
			writeLine(6, "configOverrides:")
			keys := sortedKeys(component.ConfigOverrides)
			for _, key := range keys {
				writeLine(8, fmt.Sprintf("%s: %s", key, strconv.Quote(component.ConfigOverrides[key])))
			}
		}
	}
	return buf.Bytes(), nil
}
