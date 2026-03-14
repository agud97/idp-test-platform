package load

// Cluster requirements for the default 50-environment run:
// 3 nodes x 4 vCPU x 8 GB RAM.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	defaultKubeconfig = "/root/codex/kubeconfig_6144665"
	defaultTeam       = "load"
)

type gitRepo struct {
	dir    string
	branch string
	remote string
}

type measurement struct {
	Name     string
	CommitAt time.Time
	SyncedAt time.Time
	Duration time.Duration
}

func TestConcurrency(t *testing.T) {
	ensureKubeconfig(t)

	count := envInt("LOAD_TEST_COUNT", 50)
	team := envString("LOAD_TEST_TEAM", defaultTeam)
	pushEach := envBool("LOAD_TEST_PUSH_EACH_COMMIT", true)

	repo := cloneRepo(t)
	suffix := uniqueSuffix()
	measurements := make([]measurement, 0, count)
	namespaces := make([]string, 0, count)

	startWindow := time.Now().UTC()
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("load-%02d-%s", i, suffix)
		path := filepath.Join(repo.dir, "environments", team, name+".yaml")
		repo.writeFile(t, path, manifest(name, team))
		commitAt := repo.commit(t, fmt.Sprintf("Add load-test environment %s", name))
		if pushEach {
			repo.push(t)
		}

		measurements = append(measurements, measurement{Name: name, CommitAt: commitAt})
		namespaces = append(namespaces, fmt.Sprintf("env-%s-%s", team, name))
	}
	if !pushEach {
		repo.push(t)
	}
	if time.Since(startWindow) > 5*time.Minute {
		t.Fatalf("50 sequential commits exceeded 5-minute submission window: %s", time.Since(startWindow))
	}

	t.Cleanup(func() {
		repo.cleanupBatch(t, team, measurements)
		waitForNamespacesDeleted(t, namespaces, 15*time.Minute)
	})

	deadline := 8 * time.Minute
	for i := range measurements {
		name := measurements[i].Name
		namespace := fmt.Sprintf("env-%s-%s", team, name)
		app := fmt.Sprintf("%s-%s", team, name)
		waitForApplicationAndNamespace(t, deadline, app, namespace)
		measurements[i].SyncedAt = time.Now().UTC()
		measurements[i].Duration = measurements[i].SyncedAt.Sub(measurements[i].CommitAt)
	}

	printTable(t, measurements)

	durations := make([]time.Duration, 0, len(measurements))
	var max time.Duration
	for _, result := range measurements {
		durations = append(durations, result.Duration)
		if result.Duration > max {
			max = result.Duration
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	median := durations[len(durations)/2]

	if median > 5*time.Minute {
		t.Fatalf("median sync time exceeded SLA: %s", median)
	}
	if max > 8*time.Minute {
		t.Fatalf("max sync time exceeded SLA: %s", max)
	}
}

func manifest(name, team string) string {
	return fmt.Sprintf(`apiVersion: idp.platform.io/v1alpha1
kind: Environment
metadata:
  name: %s
spec:
  owner: group:default/platform
  team: %s
  components:
    - name: web
      type: webapp
      enabled: true
      replicas: 1
`, name, team)
}

func printTable(t *testing.T, measurements []measurement) {
	t.Helper()
	var builder strings.Builder
	builder.WriteString("\nname\tcommit_at\tsynced_at\tduration\n")
	for _, item := range measurements {
		builder.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\n", item.Name, item.CommitAt.Format(time.RFC3339), item.SyncedAt.Format(time.RFC3339), item.Duration.Round(time.Second)))
	}
	t.Log(builder.String())
}

func ensureKubeconfig(t *testing.T) {
	t.Helper()
	if _, ok := os.LookupEnv("KUBECONFIG"); ok {
		return
	}
	if _, err := os.Stat(defaultKubeconfig); err != nil {
		t.Fatalf("KUBECONFIG is unset and default kubeconfig %q is unavailable: %v", defaultKubeconfig, err)
	}
	t.Setenv("KUBECONFIG", defaultKubeconfig)
}

func cloneRepo(t *testing.T) gitRepo {
	t.Helper()
	remote := strings.TrimSpace(runCmd(t, "", nil, "git", "remote", "get-url", "origin"))
	branch := strings.TrimSpace(runCmd(t, "", nil, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	dir := t.TempDir()

	runCmdRetry(t, "", nil, 3, "git", "clone", "--branch", branch, "--single-branch", remote, dir)
	runCmd(t, dir, nil, "git", "config", "user.name", "Codex Load Test")
	runCmd(t, dir, nil, "git", "config", "user.email", "codex-load@example.invalid")

	return gitRepo{dir: dir, branch: branch, remote: remote}
}

func (r gitRepo) writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func (r gitRepo) commit(t *testing.T, message string) time.Time {
	t.Helper()
	runCmd(t, r.dir, nil, "git", "add", "environments")
	runCmd(t, r.dir, nil, "git", "commit", "-m", message)
	ts := strings.TrimSpace(runCmd(t, r.dir, nil, "git", "log", "-1", "--format=%cI"))
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Fatalf("parse commit timestamp %q: %v", ts, err)
	}
	return parsed
}

func (r gitRepo) push(t *testing.T) {
	t.Helper()
	runCmdRetry(t, r.dir, nil, 3, "git", "push", "origin", r.branch)
}

func (r gitRepo) cleanupBatch(t *testing.T, team string, measurements []measurement) {
	t.Helper()
	changed := false
	for _, item := range measurements {
		path := filepath.Join(r.dir, "environments", team, item.Name+".yaml")
		if err := os.Remove(path); err == nil {
			changed = true
		}
	}
	if !changed {
		return
	}
	runCmd(t, r.dir, nil, "git", "add", "environments")
	runCmd(t, r.dir, nil, "git", "commit", "-m", "Cleanup load-test environments")
	runCmdRetry(t, r.dir, nil, 3, "git", "push", "origin", r.branch)
}

func waitForNamespacesDeleted(t *testing.T, namespaces []string, timeout time.Duration) {
	t.Helper()
	expected := make(map[string]struct{}, len(namespaces))
	for _, namespace := range namespaces {
		expected[namespace] = struct{}{}
	}
	waitFor(t, timeout, "load-test namespaces deleted", func() (bool, string) {
		existing := strings.Fields(kubectlMaybeGet("namespace", "-o", `jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}`))
		var remaining []string
		for _, namespace := range existing {
			if _, ok := expected[namespace]; ok {
				remaining = append(remaining, namespace)
			}
		}
		return len(remaining) == 0, fmt.Sprintf("remaining=%d", len(remaining))
	})
}

func waitForApplicationAndNamespace(t *testing.T, timeout time.Duration, app, namespace string) {
	t.Helper()
	script := fmt.Sprintf(`
set -eu
for i in $(seq 1 %d); do
  sync="$(kubectl --request-timeout=15s get application -n argocd %s -o jsonpath='{.status.sync.status}' 2>/dev/null || true)"
  ns="$(kubectl --request-timeout=15s get namespace %s -o name 2>/dev/null || true)"
  if [ "$sync" = "Synced" ] && [ -n "$ns" ]; then
    exit 0
  fi
  sleep 10
done
echo "sync=$sync namespace=$ns" >&2
exit 1
`, int(timeout/(10*time.Second)), shellQuote(app), shellQuote(namespace))
	runCmd(t, "", nil, "bash", "-lc", script)
}

func waitFor(t *testing.T, timeout time.Duration, description string, fn func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		ok, detail := fn()
		if ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s timed out after %s: %s", description, timeout, detail)
		}
		time.Sleep(10 * time.Second)
	}
}

func kubectlMaybeGet(args ...string) string {
	for attempt := 0; attempt < 3; attempt++ {
		output, err := runCmdCombined("", nil, "kubectl", append([]string{"--request-timeout=15s"}, args...)...)
		if err == nil {
			return strings.TrimSpace(output)
		}
		time.Sleep(2 * time.Second)
	}
	return ""
}

func runCmd(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	output, err := runCmdCombined(dir, env, name, args...)
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return output
}

func runCmdRetry(t *testing.T, dir string, env []string, attempts int, name string, args ...string) string {
	t.Helper()
	var output string
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		output, err = runCmdCombined(dir, env, name, args...)
		if err == nil {
			return output
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("%s %s failed after %d attempts: %v\n%s", name, strings.Join(args, " "), attempts, err, output)
	return ""
}

func runCmdCombined(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	combined := stdout.String()
	if stderr.Len() > 0 {
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += stderr.String()
	}
	return combined, err
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envString(name, fallback string) string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	return raw
}

func uniqueSuffix() string {
	return strings.ToLower(time.Now().UTC().Format("150405"))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
