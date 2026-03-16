package validator

import (
	"testing"

	"github.com/agud97/idp-platform/cli/pkg/converter"
	"github.com/stretchr/testify/require"
)

func TestValidateValidManifestReturnsNoErrors(t *testing.T) {
	t.Parallel()

	v := mustValidator(t)
	manifest := validManifest()

	errs := v.Validate(manifest)
	require.Empty(t, errs)
}

func TestValidateReplicasAboveMaximum(t *testing.T) {
	t.Parallel()

	v := mustValidator(t)
	manifest := validManifest()
	replicas := int32(51)
	manifest.Spec.Components[0].Replicas = &replicas

	errs := v.Validate(manifest)
	require.NotEmpty(t, errs)
	require.Contains(t, joinedMessages(errs), "0–50")
}

func TestValidateDuplicateComponentNames(t *testing.T) {
	t.Parallel()

	v := mustValidator(t)
	manifest := validManifest()
	manifest.Spec.Components = append(manifest.Spec.Components, manifest.Spec.Components[0])

	errs := v.Validate(manifest)
	require.NotEmpty(t, errs)
	require.Contains(t, joinedMessages(errs), "duplicate component name")
}

func TestValidateUnknownConfigOverrideKey(t *testing.T) {
	t.Parallel()

	v := mustValidator(t)
	manifest := validManifest()
	manifest.Spec.Components[0].ConfigOverrides["bad-key"] = "value"

	errs := v.Validate(manifest)
	require.NotEmpty(t, errs)
	require.Contains(t, joinedMessages(errs), `invalid key "bad-key"`)
}

func TestValidateSensitiveConfigOverrideKey(t *testing.T) {
	t.Parallel()

	v := mustValidator(t)
	manifest := validManifest()
	manifest.Spec.Components[0].ConfigOverrides["DB_PASSWORD"] = "secret"

	errs := v.Validate(manifest)
	require.NotEmpty(t, errs)
	require.Contains(t, joinedMessages(errs), `sensitive key "DB_PASSWORD"`)
}

func TestValidateInvalidImageTag(t *testing.T) {
	t.Parallel()

	v := mustValidator(t)
	manifest := validManifest()
	manifest.Spec.Components[0].ImageTag = "bad tag"

	errs := v.Validate(manifest)
	require.NotEmpty(t, errs)
	require.Contains(t, joinedMessages(errs), "valid OCI tag")
}

func mustValidator(t *testing.T) EnvironmentValidator {
	t.Helper()

	v, err := NewEnvironmentValidator()
	require.NoError(t, err)
	return v
}

func validManifest() *converter.EnvironmentManifest {
	replicas := int32(1)
	return &converter.EnvironmentManifest{
		APIVersion: "idp.platform.io/v1alpha1",
		Kind:       "Environment",
		Metadata: converter.Metadata{
			Name: "demo",
		},
		Spec: converter.EnvironmentSpec{
			Owner: "owner",
			Team:  "team",
			Components: []converter.Component{{
				Name:     "api",
				Type:     "webapp",
				Enabled:  true,
				ImageTag: "v1.2.3",
				Replicas: &replicas,
				ConfigOverrides: map[string]string{
					"LOG_LEVEL": "debug",
				},
			}},
		},
	}
}

func joinedMessages(errs []ValidationError) string {
	out := ""
	for _, err := range errs {
		out += err.Message + "\n"
	}
	return out
}
