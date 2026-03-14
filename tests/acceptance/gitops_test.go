package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGitOps(t *testing.T) {
	ensureKubeconfig(t)

	repo := cloneRepo(t)

	envA := "gitopsa-" + uniqueSuffix()
	envB := "gitopsb-" + uniqueSuffix()
	appA := fmt.Sprintf("%s-%s", testTeam, envA)
	nsA := fmt.Sprintf("env-%s-%s", testTeam, envA)
	nsB := fmt.Sprintf("env-%s-%s", testTeam, envB)
	pathA := filepath.Join(repo.dir, "environments", testTeam, envA+".yaml")
	pathB := filepath.Join(repo.dir, "environments", testTeam, envB+".yaml")

	t.Cleanup(func() {
		_ = repo.cleanupEnvironment(t, envA)
		_ = repo.cleanupEnvironment(t, envB)
	})

	repo.writeFile(t, pathA, gitopsManifest(envA))
	repo.writeFile(t, pathB, gitopsManifest(envB))
	repo.commitAndPush(t, "Add gitops acceptance manifests "+envA+" "+envB)

	waitFor(t, 5*time.Minute, "gitops environments ready", func() (bool, string) {
		readyA := kubectlMaybeGet("environment", "-n", controlNamespace, envA, "-o", `jsonpath={.status.conditions[?(@.type=="Ready")].status}`)
		readyB := kubectlMaybeGet("environment", "-n", controlNamespace, envB, "-o", `jsonpath={.status.conditions[?(@.type=="Ready")].status}`)
		return readyA == "True" && readyB == "True", fmt.Sprintf("readyA=%q readyB=%q", readyA, readyB)
	})

	waitFor(t, 5*time.Minute, "both web deployments ready", func() (bool, string) {
		readyA := kubectlMaybeGet("deployment", "-n", nsA, "web", "-o", `jsonpath={.status.readyReplicas}`)
		readyB := kubectlMaybeGet("deployment", "-n", nsB, "web", "-o", `jsonpath={.status.readyReplicas}`)
		return readyA == "1" && readyB == "1", fmt.Sprintf("readyA=%q readyB=%q", readyA, readyB)
	})

	t.Run("AC-037-manual-delete-restored", func(t *testing.T) {
		runCmd(t, "", nil, "kubectl", "delete", "environment", "-n", controlNamespace, envA)
		start := time.Now()
		waitFor(t, 3*time.Minute, "environment restored after delete", func() (bool, string) {
			ready := kubectlMaybeGet("environment", "-n", controlNamespace, envA, "-o", `jsonpath={.status.conditions[?(@.type=="Ready")].status}`)
			deploy := kubectlMaybeGet("deployment", "-n", nsA, "web", "-o", `jsonpath={.status.readyReplicas}`)
			return ready == "True" && deploy == "1", fmt.Sprintf("ready=%q deploy=%q", ready, deploy)
		})
		if time.Since(start) > 3*time.Minute {
			t.Fatalf("restore after delete exceeded SLA: %s", time.Since(start))
		}
	})

	t.Run("AC-038-manual-scale-reverted", func(t *testing.T) {
		patch := `{"spec":{"components":[{"name":"web","type":"webapp","enabled":true,"replicas":5,"imageRepository":"docker.io/library/nginx","imageTag":"1.27.4"}]}}`
		runCmd(t, "", nil, "kubectl", "patch", "environment", "-n", controlNamespace, envA, "--type=merge", "-p", patch)
		start := time.Now()
		waitFor(t, 3*time.Minute, "environment replicas reverted", func() (bool, string) {
			replicas := kubectlMaybeGet("environment", "-n", controlNamespace, envA, "-o", `jsonpath={.spec.components[0].replicas}`)
			deploy := kubectlMaybeGet("deployment", "-n", nsA, "web", "-o", `jsonpath={.spec.replicas}`)
			ready := kubectlMaybeGet("deployment", "-n", nsA, "web", "-o", `jsonpath={.status.readyReplicas}`)
			return replicas == "1" && deploy == "1" && ready == "1", fmt.Sprintf("claim=%q deploy=%q ready=%q", replicas, deploy, ready)
		})
		if time.Since(start) > 3*time.Minute {
			t.Fatalf("scale revert exceeded SLA: %s", time.Since(start))
		}
	})

	t.Run("AC-040-kubectl-apply-reverted", func(t *testing.T) {
		manifestPath := filepath.Join(t.TempDir(), "env-override.yaml")
		override := fmt.Sprintf(`apiVersion: idp.platform.io/v1alpha1
kind: Environment
metadata:
  name: %s
  namespace: %s
spec:
  owner: group:default/platform
  team: %s
  components:
    - name: web
      type: webapp
      enabled: true
      replicas: 0
`, envA, controlNamespace, testTeam)
		if err := os.WriteFile(manifestPath, []byte(override), 0o644); err != nil {
			t.Fatalf("write override manifest: %v", err)
		}
		runCmd(t, "", nil, "kubectl", "apply", "-f", manifestPath)

		start := time.Now()
		waitFor(t, 3*time.Minute, "kubectl apply reverted by gitops", func() (bool, string) {
			replicas := kubectlMaybeGet("environment", "-n", controlNamespace, envA, "-o", `jsonpath={.spec.components[0].replicas}`)
			deploy := kubectlMaybeGet("deployment", "-n", nsA, "web", "-o", `jsonpath={.spec.replicas}`)
			return replicas == "1" && deploy == "1", fmt.Sprintf("claimReplicas=%q deploy=%q", replicas, deploy)
		})
		if time.Since(start) > 3*time.Minute {
			t.Fatalf("apply revert exceeded SLA: %s", time.Since(start))
		}
	})

	t.Run("AC-019-AC-041-cross-namespace-denied-and-AC-042-intra-namespace-allowed", func(t *testing.T) {
		createProbePod(t, nsA, "probe-a")
		defer deleteProbePod(t, nsA, "probe-a")
		createProbePod(t, nsB, "probe-b")
		defer deleteProbePod(t, nsB, "probe-b")

		targetA := kubectlGet(t, "pod", "-n", nsA, "-l", "app.kubernetes.io/instance=web", "-o", `jsonpath={.items[0].status.podIP}`)
		targetB := kubectlGet(t, "pod", "-n", nsB, "-l", "app.kubernetes.io/instance=web", "-o", `jsonpath={.items[0].status.podIP}`)

		intraPort := ""
		for _, port := range []string{"80", "8080"} {
			out, err := runCmdCombined("", nil, "kubectl", "exec", "-n", nsA, "probe-a", "--", "sh", "-c", fmt.Sprintf("nc -zvw 5 %s %s", targetA, port))
			if err == nil && (strings.Contains(strings.ToLower(out), "succeeded") || strings.Contains(strings.ToLower(out), "open")) {
				intraPort = port
				break
			}
		}
		if intraPort == "" {
			t.Fatalf("expected intra-namespace pod-to-pod connection to succeed, but neither port 80 nor 8080 worked")
		}

		cross, err := runCmdCombined("", nil, "kubectl", "exec", "-n", nsA, "probe-a", "--", "sh", "-c", fmt.Sprintf("nc -zvw 5 %s %s", targetB, intraPort))
		if err == nil {
			t.Fatalf("expected cross-namespace connection to fail, got success:\n%s", cross)
		}
		lower := strings.ToLower(cross)
		if strings.Contains(lower, "connection reset") {
			t.Fatalf("unexpected connection reset; expected timeout or refusal:\n%s", cross)
		}
		if !strings.Contains(lower, "timed out") && !strings.Contains(lower, "refused") && !strings.Contains(lower, "operation not permitted") {
			t.Fatalf("expected timeout, refusal, or policy denial for cross-namespace connection, got:\n%s", cross)
		}
		t.Logf("cross-namespace dial result: %s", strings.TrimSpace(cross))
	})

	t.Run("AC-039-git-history-recorded", func(t *testing.T) {
		log := runCmd(t, repo.dir, nil, "git", "log", "--format=%an <%ae> %s", "-n", "3")
		if !strings.Contains(log, "Codex Acceptance <codex@example.invalid>") {
			t.Fatalf("git history missing expected author:\n%s", log)
		}

		show := runCmd(t, repo.dir, nil, "git", "show", "--stat", "--oneline", "HEAD")
		if !strings.Contains(show, "environments/acc/") {
			t.Fatalf("git diff/stat missing expected environment manifest changes:\n%s", show)
		}
		if kubectlMaybeGet("application", "-n", "argocd", appA, "-o", "name") == "" {
			t.Fatalf("expected generated Application %q to exist", appA)
		}
	})
}

func ensureKubeconfig(t *testing.T) {
	t.Helper()
	if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
		if _, err := os.Stat(defaultKubeconfig); err != nil {
			t.Fatalf("KUBECONFIG is unset and default kubeconfig %q is unavailable: %v", defaultKubeconfig, err)
		}
		t.Setenv("KUBECONFIG", defaultKubeconfig)
	}
}

func gitopsManifest(name string) string {
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
      imageRepository: docker.io/library/nginx
      imageTag: 1.27.4
`, name, testTeam)
}

func createProbePod(t *testing.T, namespace, name string) {
	t.Helper()
	runCmd(t, "", nil, "kubectl", "run", name, "-n", namespace, "--image=busybox:1.36", "--restart=Never", "--command", "--", "sh", "-c", "sleep 600")
	waitFor(t, 2*time.Minute, "probe pod ready", func() (bool, string) {
		phase := kubectlMaybeGet("pod", "-n", namespace, name, "-o", `jsonpath={.status.phase}`)
		return phase == "Running", fmt.Sprintf("phase=%q", phase)
	})
}

func deleteProbePod(t *testing.T, namespace, name string) {
	t.Helper()
	_, _ = runCmdCombined("", nil, "kubectl", "delete", "pod", "-n", namespace, name, "--ignore-not-found=true")
}
