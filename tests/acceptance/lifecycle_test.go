package acceptance

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	controlNamespace = "crossplane-system"
	testTeam         = "acc"
	pollInterval     = 5 * time.Second
	defaultKubeconfig = "/root/codex/kubeconfig_6144665"
)

type gitRepo struct {
	dir    string
	branch string
	remote string
}

func TestLifecycle(t *testing.T) {
	if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
		if _, err := os.Stat(defaultKubeconfig); err != nil {
			t.Fatalf("KUBECONFIG is unset and default kubeconfig %q is unavailable: %v", defaultKubeconfig, err)
		}
		t.Setenv("KUBECONFIG", defaultKubeconfig)
	}

	repo := cloneRepo(t)
	t.Cleanup(func() {
		_ = repo.cleanupEnvironment(t, "invalid-"+uniqueSuffix())
	})

	validName := "life-" + uniqueSuffix()
	validApp := fmt.Sprintf("%s-%s", testTeam, validName)
	validNamespace := fmt.Sprintf("env-%s-%s", testTeam, validName)
	validPath := filepath.Join(repo.dir, "environments", testTeam, validName+".yaml")

	t.Run("AC-001-create-valid-manifest", func(t *testing.T) {
		start := time.Now()
		repo.writeFile(t, validPath, validManifest(validName, true, "# create"))
		repo.commitAndPush(t, "Add lifecycle acceptance manifest "+validName)

		waitFor(t, 5*time.Minute, "environment ready", func() (bool, string) {
			synced := kubectlMaybeGet("environment", "-n", controlNamespace, validName, "-o", `jsonpath={.status.conditions[?(@.type=="Synced")].status}`)
			ready := kubectlMaybeGet("environment", "-n", controlNamespace, validName, "-o", `jsonpath={.status.conditions[?(@.type=="Ready")].status}`)
			if synced == "True" && ready == "True" {
				return true, "Environment claim is Synced and Ready"
			}
			return false, fmt.Sprintf("synced=%q ready=%q", synced, ready)
		})

		waitFor(t, 5*time.Minute, "workload deployment ready", func() (bool, string) {
			if kubectlMaybeGet("namespace", validNamespace, "-o", "name") == "" {
				return false, "namespace not created yet"
			}
			ready := kubectlGet(t, "deployment", "-n", validNamespace, "web", "-o", `jsonpath={.status.readyReplicas}`)
			if ready == "1" {
				return true, "deployment/web has 1 ready replica"
			}
			return false, "deployment/web not ready yet"
		})

		elapsed := time.Since(start)
		t.Logf("AC-001 elapsed seconds: %.0f", elapsed.Seconds())
		if elapsed > 5*time.Minute {
			t.Fatalf("environment exceeded SLA: %s", elapsed)
		}
	})

	t.Run("AC-007-idempotent-reapply", func(t *testing.T) {
		beforeGeneration := kubectlGet(t, "deployment", "-n", validNamespace, "web", "-o", `jsonpath={.metadata.generation}`)
		beforePod := kubectlGet(t, "pod", "-n", validNamespace, "-l", "app.kubernetes.io/instance=web", "-o", `jsonpath={.items[0].metadata.name}`)

		repo.writeFile(t, validPath, validManifest(validName, true, "# idempotent-reapply"))
		repo.commitAndPush(t, "Reapply lifecycle manifest "+validName)
		waitFor(t, 45*time.Second, "comment-only reapply observed", func() (bool, string) {
			app := kubectlMaybeGet("application", "-n", "argocd", validApp, "-o", `jsonpath={.metadata.name}`)
			return app == validApp, fmt.Sprintf("application=%q", app)
		})

		afterGeneration := kubectlGet(t, "deployment", "-n", validNamespace, "web", "-o", `jsonpath={.metadata.generation}`)
		afterPod := kubectlGet(t, "pod", "-n", validNamespace, "-l", "app.kubernetes.io/instance=web", "-o", `jsonpath={.items[0].metadata.name}`)

		if beforeGeneration != afterGeneration {
			t.Fatalf("deployment generation changed on idempotent reapply: before=%s after=%s", beforeGeneration, afterGeneration)
		}
		if beforePod != afterPod {
			t.Fatalf("pod changed on idempotent reapply: before=%s after=%s", beforePod, afterPod)
		}
	})

	t.Run("AC-005-disable-component-and-AC-008-preserve-manifest", func(t *testing.T) {
		start := time.Now()
		repo.writeFile(t, validPath, validManifest(validName, false, "# disable"))
		repo.commitAndPush(t, "Disable lifecycle component "+validName)

		waitFor(t, 3*time.Minute, "deployment removed after disable", func() (bool, string) {
			if kubectlMaybeGet("deployment", "-n", validNamespace, "web", "-o", "name") == "" {
				return true, "deployment/web deleted"
			}
			return false, "deployment/web still exists"
		})

		if time.Since(start) > 3*time.Minute {
			t.Fatalf("disable exceeded SLA: %s", time.Since(start))
		}

		data, err := os.ReadFile(validPath)
		if err != nil {
			t.Fatalf("read manifest after disable: %v", err)
		}
		if !strings.Contains(string(data), "enabled: false") {
			t.Fatalf("manifest was not preserved with enabled:false")
		}
	})

	t.Run("AC-006-delete-manifest", func(t *testing.T) {
		start := time.Now()
		if err := os.Remove(validPath); err != nil {
			t.Fatalf("remove manifest: %v", err)
		}
		repo.commitAndPush(t, "Delete lifecycle manifest "+validName)

		waitFor(t, 5*time.Minute, "namespace pruned after delete", func() (bool, string) {
			app := kubectlMaybeGet("application", "-n", "argocd", validApp, "-o", "name")
			ns := kubectlMaybeGet("namespace", validNamespace, "-o", "name")
			if app == "" && ns == "" {
				return true, "application and namespace removed"
			}
			return false, fmt.Sprintf("application=%q namespace=%q", app, ns)
		})

		if time.Since(start) > 5*time.Minute {
			t.Fatalf("delete exceeded SLA: %s", time.Since(start))
		}
	})

	t.Run("AC-003-invalid-schema", func(t *testing.T) {
		invalidName := "invalid-" + uniqueSuffix()
		invalidApp := fmt.Sprintf("%s-%s", testTeam, invalidName)
		invalidNamespace := fmt.Sprintf("env-%s-%s", testTeam, invalidName)
		invalidPath := filepath.Join(repo.dir, "environments", testTeam, invalidName+".yaml")

		t.Cleanup(func() {
			repo.cleanupEnvironment(t, invalidName)
		})

		repo.writeFile(t, invalidPath, invalidManifest(invalidName))
		repo.commitAndPush(t, "Add invalid lifecycle manifest "+invalidName)

		waitFor(t, 3*time.Minute, "application degraded on invalid schema", func() (bool, string) {
			sync := kubectlMaybeGet("application", "-n", "argocd", invalidApp, "-o", `jsonpath={.status.sync.status}`)
			resourceStatus := kubectlMaybeGet("application", "-n", "argocd", invalidApp, "-o", `jsonpath={.status.operationState.syncResult.resources[0].status}`)
			if sync == "OutOfSync" && resourceStatus == "SyncFailed" {
				return true, "application reports OutOfSync with SyncFailed resource"
			}
			return false, fmt.Sprintf("sync=%q resourceStatus=%q", sync, resourceStatus)
		})

		if kubectlMaybeGet("environment", "-n", controlNamespace, invalidName, "-o", "name") != "" {
			t.Fatalf("invalid manifest unexpectedly created Environment claim")
		}
		if kubectlMaybeGet("namespace", invalidNamespace, "-o", "name") != "" {
			t.Fatalf("invalid manifest unexpectedly created workload namespace")
		}
	})
}

func cloneRepo(t *testing.T) gitRepo {
	t.Helper()
	remote := strings.TrimSpace(runCmd(t, "", nil, "git", "remote", "get-url", "origin"))
	branch := strings.TrimSpace(runCmd(t, "", nil, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	dir := t.TempDir()

	runCmdRetry(t, "", nil, 3, "git", "clone", "--branch", branch, "--single-branch", remote, dir)
	runCmd(t, dir, nil, "git", "config", "user.name", "Codex Acceptance")
	runCmd(t, dir, nil, "git", "config", "user.email", "codex@example.invalid")

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

func (r gitRepo) commitAndPush(t *testing.T, message string) {
	t.Helper()
	runCmd(t, r.dir, nil, "git", "add", "environments")
	runCmd(t, r.dir, nil, "git", "commit", "-m", message)
	runCmdRetry(t, r.dir, nil, 3, "git", "push", "origin", r.branch)
}

func (r gitRepo) cleanupEnvironment(t *testing.T, envName string) error {
	t.Helper()
	path := filepath.Join(r.dir, "environments", testTeam, envName+".yaml")
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return err
		}
		if err := runCmdErr(r.dir, nil, "git", "add", "environments"); err != nil {
			return err
		}
		if err := runCmdErr(r.dir, nil, "git", "commit", "-m", "Cleanup acceptance manifest "+envName); err != nil {
			if !strings.Contains(err.Error(), "nothing to commit") {
				return err
			}
		}
		if err := runCmdErr(r.dir, nil, "git", "push", "origin", r.branch); err != nil {
			return err
		}
	}
	return nil
}

func validManifest(name string, enabled bool, comment string) string {
	return fmt.Sprintf(`%s
apiVersion: idp.platform.io/v1alpha1
kind: Environment
metadata:
  name: %s
spec:
  owner: group:default/platform
  team: %s
  components:
    - name: web
      type: webapp
      enabled: %t
      imageRepository: docker.io/library/nginx
      imageTag: 1.27.4
`, comment, name, testTeam, enabled)
}

func invalidManifest(name string) string {
	return fmt.Sprintf(`apiVersion: idp.platform.io/v1alpha1
kind: Environment
metadata:
  name: %s
spec:
  owner: group:default/platform
  team: %s
  components:
    - name: dup
      type: webapp
      enabled: true
      imageRepository: docker.io/library/nginx
      imageTag: 1.27.4
    - name: dup
      type: webapp
      enabled: true
      imageRepository: docker.io/library/nginx
      imageTag: 1.27.4
`, name, testTeam)
}

func uniqueSuffix() string {
	return time.Now().UTC().Format("150405")
}

func waitFor(t *testing.T, timeout time.Duration, label string, cond func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		ok, msg := cond()
		last = msg
		if ok {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("%s timed out after %s: %s", label, timeout, last)
}

func kubectlGet(t *testing.T, args ...string) string {
	t.Helper()
	all := append([]string{}, args...)
	out := runCmd(t, "", nil, "kubectl", append([]string{"get"}, all...)...)
	return strings.TrimSpace(out)
}

func kubectlMaybeGet(args ...string) string {
	all := append([]string{"get"}, args...)
	cmd := exec.Command("kubectl", all...)
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}

func runCmd(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	out, err := runCmdCombined(dir, env, name, args...)
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return out
}

func runCmdRetry(t *testing.T, dir string, env []string, attempts int, name string, args ...string) string {
	t.Helper()
	var out string
	var err error
	for i := 0; i < attempts; i++ {
		out, err = runCmdCombined(dir, env, name, args...)
		if err == nil {
			return out
		}
		if i < attempts-1 {
			time.Sleep(2 * time.Second)
		}
	}
	t.Fatalf("%s %s failed after %d attempts: %v\n%s", name, strings.Join(args, " "), attempts, err, out)
	return ""
}

func runCmdErr(dir string, env []string, name string, args ...string) error {
	_, err := runCmdCombined(dir, env, name, args...)
	return err
}

func runCmdCombined(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if serr := strings.TrimSpace(stderr.String()); serr != "" {
		if out != "" {
			out += "\n"
		}
		out += serr
	}
	return out, err
}
