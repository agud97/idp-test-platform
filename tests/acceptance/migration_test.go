package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMigration(t *testing.T) {
	cliBin := buildCLI(t)

	t.Run("AC-052-AC-053-AC-055-migrate-compose", func(t *testing.T) {
		tmpDir := t.TempDir()
		composePath := filepath.Join(tmpDir, "docker-compose.yaml")
		outputPath := filepath.Join(tmpDir, "environment.yaml")
		source := []byte(`services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: platform
      POSTGRES_PASSWORD: supersecret
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
  frontend:
    image: ghcr.io/example/frontend:v1.4.0
    ports:
      - "3000:3000"
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
  worker-backend:
    image: org/backend:v5
    environment:
      QUEUE_NAME: jobs
      WORKER_KEY: abc123
  admin-api:
    image: org/api:v9
    ports:
      - "9000:9000"
  mystery:
    image: busybox:latest
    command: ["sleep", "3600"]
`)
		if err := os.WriteFile(composePath, source, 0o644); err != nil {
			t.Fatalf("write compose fixture: %v", err)
		}

		stdout := runCLI(t, cliBin, nil, "migrate", "--from", composePath, "--output", outputPath)
		if !strings.Contains(stdout, "MappedServices:") {
			t.Fatalf("expected conversion report in stdout, got:\n%s", stdout)
		}
		if !strings.Contains(stdout, `TODO no matching component type found`) {
			t.Fatalf("expected manual-review TODO for unknown service, got:\n%s", stdout)
		}

		rendered, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("read migrated output: %v", err)
		}
		if count := strings.Count(string(rendered), "enabled: true"); count < 9 {
			t.Fatalf("expected at least 9 mapped services, got %d\n%s", count, rendered)
		}
		if !strings.Contains(string(rendered), "name: mystery") || !strings.Contains(string(rendered), "enabled: false") {
			t.Fatalf("expected unknown service to be preserved as disabled in output:\n%s", rendered)
		}

		after, err := os.ReadFile(composePath)
		if err != nil {
			t.Fatalf("re-read source compose: %v", err)
		}
		if string(after) != string(source) {
			t.Fatal("source compose file was modified by migrate command")
		}
	})

	t.Run("AC-059-export-redaction", func(t *testing.T) {
		out := runCmd(t, cliDir(t), nil, goTool(t), "test", "./pkg/exporter", "-run", "TestExportRedactsSensitiveVariables", "-count=1", "-v")
		if !strings.Contains(out, "--- PASS: TestExportRedactsSensitiveVariables") {
			t.Fatalf("expected exporter redaction test to pass, got:\n%s", out)
		}
	})

	t.Run("AC-073-AC-080-migration-lifecycle", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "tracker.db")
		env := map[string]string{
			"IDP_TRACKER_DB": dbPath,
			"IDP_REPORT_DIR": tmpDir,
		}

		trackerHelper(t, env, "create-team", "alpha")

		stdout := runCLI(t, cliBin, env, "register-legacy", "--name", "shop", "--team", "alpha")
		if !strings.Contains(stdout, `Registered legacy environment "shop"`) {
			t.Fatalf("expected successful registration output, got:\n%s", stdout)
		}

		shopID, shopStatus, shopCompletedAt := findLegacyEnv(t, env, "alpha", "shop")
		if shopStatus != "pending" || shopCompletedAt != "" {
			t.Fatalf("unexpected shop state after register: status=%q completed_at=%q", shopStatus, shopCompletedAt)
		}

		if out, err := runCLICombined(cliBin, env, "register-legacy", "--name", "ghost", "--team", "missing"); err == nil || !strings.Contains(out, `Team "missing" not found`) {
			t.Fatalf("expected team-not-found error, got output=%q err=%v", out, err)
		}

		if out, err := runCLICombined(cliBin, env, "register-legacy", "--name", "shop", "--team", "alpha"); err == nil || !strings.Contains(out, `already registered`) {
			t.Fatalf("expected duplicate registration error, got output=%q err=%v", out, err)
		}

		if out, err := runCLICombined(cliBin, env, "complete-migration", "--env-id", strconv.FormatInt(shopID, 10)); err == nil || !strings.Contains(out, `status "pending"`) {
			t.Fatalf("expected pending-status completion error, got output=%q err=%v", out, err)
		}

		trackerHelper(t, env, "set-status", strconv.FormatInt(shopID, 10), "validated")

		stdout = runCLI(t, cliBin, env, "complete-migration", "--env-id", strconv.FormatInt(shopID, 10))
		if !strings.Contains(stdout, "Migration completed") {
			t.Fatalf("expected successful completion output, got:\n%s", stdout)
		}

		_, shopStatus, shopCompletedAt = findLegacyEnv(t, env, "alpha", "shop")
		if shopStatus != "completed" || shopCompletedAt == "" {
			t.Fatalf("unexpected shop state after completion: status=%q completed_at=%q", shopStatus, shopCompletedAt)
		}

		runCLI(t, cliBin, env, "register-legacy", "--name", "cart", "--team", "alpha")
		cartID, cartStatus, _ := findLegacyEnv(t, env, "alpha", "cart")
		if cartStatus != "pending" {
			t.Fatalf("unexpected cart state after register: status=%q", cartStatus)
		}

		blockedOut, err := runCLICombined(cliBin, env, "deprecate-legacy", "--dry-run")
		if err == nil || !strings.Contains(blockedOut, "deprecation gate blocked") {
			t.Fatalf("expected deprecation gate block, got output=%q err=%v", blockedOut, err)
		}
		if !strings.Contains(blockedOut, "Gate FAILED") || !strings.Contains(blockedOut, "cart | alpha | pending") {
			t.Fatalf("expected blocked dry-run summary, got:\n%s", blockedOut)
		}

		trackerHelper(t, env, "set-status", strconv.FormatInt(cartID, 10), "validated")
		runCLI(t, cliBin, env, "complete-migration", "--env-id", strconv.FormatInt(cartID, 10))

		reportOut := runCLI(t, cliBin, env, "deprecate-legacy", "--confirm")
		if !strings.Contains(reportOut, "Deprecation report written") {
			t.Fatalf("expected deprecation report output, got:\n%s", reportOut)
		}

		flag := trackerHelper(t, env, "flag")
		if flag != "true" {
			t.Fatalf("expected deprecated flag to be true, got %q", flag)
		}

		reportPath := filepath.Join(tmpDir, fmt.Sprintf("deprecation-report-%s.md", time.Now().UTC().Format("2006-01-02")))
		reportBody, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read deprecation report: %v", err)
		}
		if !strings.Contains(string(reportBody), fmt.Sprintf("| %d | alpha | shop | %s |", shopID, shopCompletedAt)) {
			t.Fatalf("shop entry missing from deprecation report:\n%s", reportBody)
		}
		_, _, cartCompletedAt := findLegacyEnv(t, env, "alpha", "cart")
		if !strings.Contains(string(reportBody), fmt.Sprintf("| %d | alpha | cart | %s |", cartID, cartCompletedAt)) {
			t.Fatalf("cart entry missing from deprecation report:\n%s", reportBody)
		}

		emptyDir := t.TempDir()
		emptyEnv := map[string]string{
			"IDP_TRACKER_DB": filepath.Join(emptyDir, "tracker.db"),
			"IDP_REPORT_DIR": emptyDir,
		}
		if out, err := runCLICombined(cliBin, emptyEnv, "deprecate-legacy"); err == nil || !strings.Contains(out, "No legacy environments registered") {
			t.Fatalf("expected no-environments error, got output=%q err=%v", out, err)
		}
	})
}

func buildCLI(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "idp")
	runCmd(t, cliDir(t), nil, goTool(t), "build", "-o", binPath, "./cmd/idp")
	return binPath
}

func runCLI(t *testing.T, cliBin string, env map[string]string, args ...string) string {
	t.Helper()
	return runCmd(t, "", envSlice(env), cliBin, args...)
}

func runCLICombined(cliBin string, env map[string]string, args ...string) (string, error) {
	return runCmdCombined("", envSlice(env), cliBin, args...)
}

func findLegacyEnv(t *testing.T, env map[string]string, team, name string) (int64, string, string) {
	t.Helper()
	out := trackerHelper(t, env, "find-legacy", team, name)
	parts := strings.Split(out, "|")
	if len(parts) != 3 {
		t.Fatalf("unexpected legacy environment payload %q", out)
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		t.Fatalf("parse legacy environment id %q: %v", parts[0], err)
	}
	return id, parts[1], parts[2]
}

func trackerHelper(t *testing.T, env map[string]string, args ...string) string {
	t.Helper()
	helperDir := filepath.Join(cliDir(t), ".acceptance-helper-"+uniqueSuffix())
	if err := os.MkdirAll(helperDir, 0o755); err != nil {
		t.Fatalf("create helper dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(helperDir)
	})

	helperPath := filepath.Join(helperDir, "main.go")
	helper := `package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/agud97/idp-platform/cli/internal/tracker"
)

func main() {
	if len(os.Args) < 2 {
		fail(fmt.Errorf("missing action"))
	}

	dbPath := os.Getenv("IDP_TRACKER_DB")
	store, err := tracker.Open(dbPath)
	if err != nil {
		fail(err)
	}
	defer store.Close()

	switch os.Args[1] {
	case "create-team":
		team, err := store.CreateTeam(os.Args[2])
		if err != nil {
			fail(err)
		}
		fmt.Print(team.ID)
	case "find-legacy":
		team, err := store.FindTeamByName(os.Args[2])
		if err != nil {
			fail(err)
		}
		if team == nil {
			fail(fmt.Errorf("team %q not found", os.Args[2]))
		}
		record, err := store.FindLegacyEnvironmentByName(team.ID, os.Args[3])
		if err != nil {
			fail(err)
		}
		if record == nil {
			fmt.Print("0||")
			return
		}
		completedAt := ""
		if record.CompletedAt.Valid {
			completedAt = record.CompletedAt.Time.UTC().Format(time.RFC3339)
		}
		fmt.Printf("%d|%s|%s", record.ID, record.MigrationStatus, completedAt)
	case "set-status":
		id, err := strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			fail(err)
		}
		if err := store.SetMigrationStatus(id, os.Args[3]); err != nil {
			fail(err)
		}
	case "flag":
		value, err := store.DeprecatedFlag()
		if err != nil {
			fail(err)
		}
		fmt.Print(value)
	default:
		fail(fmt.Errorf("unknown action %q", os.Args[1]))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
`
	if err := os.WriteFile(helperPath, []byte(helper), 0o644); err != nil {
		t.Fatalf("write helper program: %v", err)
	}

	return strings.TrimSpace(runCmd(t, cliDir(t), envSlice(env), goTool(t), append([]string{"run", "./" + filepath.Base(helperDir)}, args...)...))
}

func envSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}
	return out
}

func cliDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "cli"))
}

func goTool(t *testing.T) string {
	t.Helper()
	if _, err := os.Stat("/tmp/go/bin/go"); err == nil {
		return "/tmp/go/bin/go"
	}
	return "go"
}
