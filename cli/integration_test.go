//go:build integration

package integration

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/agud97/idp-platform/cli/internal/cliapp"
	"github.com/agud97/idp-platform/cli/internal/tracker"
	"github.com/agud97/idp-platform/cli/pkg/converter"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestIntegrationFullPipeline(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "out.yaml")

	stdout, _, err := runCommand(t, cliapp.NewMigrateCommand(testLogger()), "--from", "testdata/fixtures/simple-compose.yaml", "--output", outputPath)
	require.NoError(t, err)
	require.Contains(t, stdout, "MappedServices:")

	actual, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	expectedBody, err := os.ReadFile("testdata/expected/simple-compose-expected.yaml")
	require.NoError(t, err)

	expected := append([]byte("# Source: idp-migrate\n"), expectedBody...)
	require.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(string(actual)))

	stdout, _, err = runCommand(t, cliapp.NewValidateCommand(testLogger()), "--file", outputPath)
	require.NoError(t, err)
	require.Equal(t, "Valid\n", stdout)
}

func TestIntegrationMappingAccuracy(t *testing.T) {
	t.Parallel()

rawCompose := []byte(`
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: platform
      POSTGRES_PASSWORD: supersecret
    volumes:
      - pg-data:/var/lib/postgresql/data
  redis:
    image: redis:7
    environment:
      REDIS_PASSWORD: redis-secret
  web:
    image: nginx:1.27
    ports:
      - "8080:80"
    environment:
      APP_ENV: prod
      API_TOKEN: secret-token
    volumes:
      - web-data:/usr/share/nginx/html
  frontend:
    image: ghcr.io/example/frontend:v1.4.0
    ports:
      - "3000:3000"
    environment:
      LOG_LEVEL: debug
  api:
    image: ghcr.io/example/api:v2.1.0
    ports:
      - "8081:8081"
    environment:
      APP_NAME: public-api
      DB_HOST: postgres
  backend:
    image: company/backend:3.2.1
    environment:
      FEATURE_FLAG: enabled
      CACHE_SECRET: keep-me
  gateway:
    image: library/nginx:1.25
    ports:
      - "8082:80"
    environment:
      APP_ENV: gateway
  worker-backend:
    image: org/backend:v5
    environment:
      QUEUE_NAME: jobs
      WORKER_KEY: abc123
  admin-api:
    image: org/api:v9
    ports:
      - "9000:9000"
    environment:
      LOG_LEVEL: info
  mystery:
    image: busybox:latest
    command: ["sleep", "3600"]
volumes:
  web-data: {}
  pg-data: {}
`)

	project := mustLoadProject(t, rawCompose)
	manifest, report, err := converter.NewComposeConverter().Convert(bytes.NewReader(rawCompose))
	require.NoError(t, err)
	require.NotNil(t, report)

	components := make(map[string]converter.Component, len(manifest.Spec.Components))
	componentOrder := make(map[string]int, len(manifest.Spec.Components))
	for i, component := range manifest.Spec.Components {
		components[component.Name] = component
		componentOrder[component.Name] = i
	}

	expectedTypes := map[string]string{
		"web":       "webapp",
		"frontend":  "webapp",
		"api":       "webapp",
		"backend":   "webapp",
		"postgres":  "postgresql",
		"redis":     "redis",
		"gateway":   "webapp",
		"worker-backend": "webapp",
		"admin-api": "webapp",
	}

	var correctlyMapped int
	var failures []string

	for _, service := range project.Services {
		expectedType, shouldCount := expectedTypes[service.Name]
		component, exists := components[service.Name]
		if !shouldCount {
			if exists && component.Enabled {
				failures = append(failures, fmt.Sprintf("%s: expected unmapped service to be disabled", service.Name))
			}
			continue
		}
		if !exists {
			failures = append(failures, fmt.Sprintf("%s: component missing from manifest", service.Name))
			continue
		}

		var serviceFailures []string
		if component.Type != expectedType {
			serviceFailures = append(serviceFailures, fmt.Sprintf("type=%s", component.Type))
		}
		if component.ImageTag != expectedImageTag(service.Image) {
			serviceFailures = append(serviceFailures, fmt.Sprintf("imageTag=%s", component.ImageTag))
		}
		if !sameInt32s(report.ServicePorts[service.Name], expectedPorts(service)) {
			serviceFailures = append(serviceFailures, fmt.Sprintf("ports=%v", report.ServicePorts[service.Name]))
		}
		for key, value := range service.Environment {
			actualValue, ok := component.ConfigOverrides[key]
			if !ok {
				serviceFailures = append(serviceFailures, fmt.Sprintf("missing env key %s", key))
				continue
			}
			expectedValue := ""
			if value != nil {
				expectedValue = *value
			}
			if actualValue != expectedValue {
				serviceFailures = append(serviceFailures, fmt.Sprintf("env %s=%s", key, actualValue))
			}
		}
		if !sameStrings(report.ServiceVolumes[service.Name], expectedVolumes(service)) {
			serviceFailures = append(serviceFailures, fmt.Sprintf("volumes=%v", report.ServiceVolumes[service.Name]))
		}
		for _, dependency := range expectedDependencies(service) {
			if componentOrder[dependency] > componentOrder[service.Name] {
				serviceFailures = append(serviceFailures, fmt.Sprintf("dependency order %s after %s", dependency, service.Name))
			}
		}

		if len(serviceFailures) > 0 {
			failures = append(failures, fmt.Sprintf("%s: %s", service.Name, strings.Join(serviceFailures, ", ")))
			continue
		}
		correctlyMapped++
	}

	require.GreaterOrEqualf(t, correctlyMapped, 9, "correctly mapped=%d failures=%v", correctlyMapped, failures)
	require.Emptyf(t, failures, "mapping failures: %v", failures)
}

func TestIntegrationRetentionExpiry(t *testing.T) {
	t.Setenv("LOG_RETENTION_DAYS", "0")

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "events.db"))
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, createEventStore(db))
	require.NoError(t, insertEvent(db, "old-event", time.Now().UTC().Add(-24*time.Hour)))

	events, err := retainedEvents(db, time.Now().UTC())
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestIntegrationMigrationLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "tracker.db")

	t.Setenv("IDP_TRACKER_DB", dbPath)
	t.Setenv("IDP_REPORT_DIR", tmpDir)

	store, err := tracker.Open(dbPath)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.CreateTeam("alpha")
	require.NoError(t, err)

	stdout, _, err := runCommand(t, cliapp.NewRegisterLegacyCommand(testLogger()), "--name", "shop", "--team", "alpha")
	require.NoError(t, err)
	require.Contains(t, stdout, `Registered legacy environment "shop"`)

	record, err := store.FindLegacyEnvironmentByName(1, "shop")
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "pending", record.MigrationStatus)

	_, _, err = runCommand(t, cliapp.NewCompleteMigrationCommand(testLogger()), "--env-id", fmt.Sprintf("%d", record.ID))
	require.Error(t, err)
	require.Contains(t, err.Error(), `status "pending"`)

	require.NoError(t, store.SetMigrationStatus(record.ID, "validated"))

	stdout, _, err = runCommand(t, cliapp.NewCompleteMigrationCommand(testLogger()), "--env-id", fmt.Sprintf("%d", record.ID))
	require.NoError(t, err)
	require.Contains(t, stdout, "Migration completed")

	stdout, _, err = runCommand(t, cliapp.NewDeprecateLegacyCommand(testLogger()), "--dry-run")
	require.NoError(t, err)
	require.Contains(t, stdout, "Gate PASSED")

	stdout, _, err = runCommand(t, cliapp.NewDeprecateLegacyCommand(testLogger()), "--confirm")
	require.NoError(t, err)
	require.Contains(t, stdout, "Deprecation report written")

	flag, err := store.DeprecatedFlag()
	require.NoError(t, err)
	require.Equal(t, "true", flag)

	reportPath := filepath.Join(tmpDir, fmt.Sprintf("deprecation-report-%s.md", time.Now().UTC().Format("2006-01-02")))
	report, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	require.Contains(t, string(report), "| alpha | shop |")
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func runCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func mustLoadProject(t *testing.T, raw []byte) *types.Project {
	t.Helper()

	project, err := loader.Load(types.ConfigDetails{
		WorkingDir:  ".",
		Environment: map[string]string{},
		ConfigFiles: []types.ConfigFile{{
			Filename: "docker-compose.yaml",
			Content:  raw,
		}},
	}, func(options *loader.Options) {
		options.SetProjectName("integration", true)
	})
	require.NoError(t, err)
	return project
}

func expectedImageTag(image string) string {
	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")
	if lastColon > lastSlash {
		return image[lastColon+1:]
	}
	return ""
}

func expectedPorts(service types.ServiceConfig) []int32 {
	var ports []int32
	for _, port := range service.Ports {
		if port.Target > 0 {
			ports = append(ports, int32(port.Target))
		}
	}
	return ports
}

func expectedVolumes(service types.ServiceConfig) []string {
	var volumes []string
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

func expectedDependencies(service types.ServiceConfig) []string {
	dependencies := make([]string, 0, len(service.DependsOn))
	for name := range service.DependsOn {
		dependencies = append(dependencies, name)
	}
	return dependencies
}

func sameInt32s(left, right []int32) bool {
	return slices.Equal(left, right)
}

func sameStrings(left, right []string) bool {
	return slices.Equal(left, right)
}

func createEventStore(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE migration_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	return err
}

func insertEvent(db *sql.DB, message string, createdAt time.Time) error {
	_, err := db.Exec(`INSERT INTO migration_events(message, created_at) VALUES (?, ?)`, message, createdAt.Format(time.RFC3339))
	return err
}

func retainedEvents(db *sql.DB, now time.Time) ([]string, error) {
	retentionDays := 7
	if raw := os.Getenv("LOG_RETENTION_DAYS"); raw != "" {
		_, err := fmt.Sscanf(raw, "%d", &retentionDays)
		if err != nil {
			return nil, err
		}
	}

	cutoff := now.AddDate(0, 0, -retentionDays)
	rows, err := db.Query(`SELECT message, created_at FROM migration_events`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kept []string
	for rows.Next() {
		var message string
		var createdAtRaw string
		if err := rows.Scan(&message, &createdAtRaw); err != nil {
			return nil, err
		}
		createdAt, err := time.Parse(time.RFC3339, createdAtRaw)
		if err != nil {
			return nil, err
		}
		if !createdAt.Before(cutoff) {
			kept = append(kept, message)
		}
	}
	return kept, rows.Err()
}
