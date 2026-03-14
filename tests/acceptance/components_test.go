package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestComponents(t *testing.T) {
	if _, ok := os.LookupEnv("KUBECONFIG"); !ok {
		if _, err := os.Stat(defaultKubeconfig); err != nil {
			t.Fatalf("KUBECONFIG is unset and default kubeconfig %q is unavailable: %v", defaultKubeconfig, err)
		}
		t.Setenv("KUBECONFIG", defaultKubeconfig)
	}

	repo := cloneRepo(t)

	validName := "comp-" + uniqueSuffix()
	validApp := fmt.Sprintf("%s-%s", testTeam, validName)
	validNamespace := fmt.Sprintf("env-%s-%s", testTeam, validName)
	validPath := filepath.Join(repo.dir, "environments", testTeam, validName+".yaml")

	invalidName := "replicas-" + uniqueSuffix()
	invalidApp := fmt.Sprintf("%s-%s", testTeam, invalidName)
	invalidNamespace := fmt.Sprintf("env-%s-%s", testTeam, invalidName)
	invalidPath := filepath.Join(repo.dir, "environments", testTeam, invalidName+".yaml")

	t.Cleanup(func() {
		_ = repo.cleanupEnvironment(t, validName)
		_ = repo.cleanupEnvironment(t, invalidName)
	})

	t.Run("AC-004-AC-009-AC-010-AC-015-AC-018", func(t *testing.T) {
		start := time.Now()
		repo.writeFile(t, validPath, componentsManifest(validName, 3, true))
		repo.commitAndPush(t, "Add components acceptance manifest "+validName)

		waitFor(t, 5*time.Minute, "components environment ready", func() (bool, string) {
			synced := kubectlMaybeGet("environment", "-n", controlNamespace, validName, "-o", `jsonpath={.status.conditions[?(@.type=="Synced")].status}`)
			ready := kubectlMaybeGet("environment", "-n", controlNamespace, validName, "-o", `jsonpath={.status.conditions[?(@.type=="Ready")].status}`)
			if synced == "True" && ready == "True" {
				return true, "Environment claim is Synced and Ready"
			}
			return false, fmt.Sprintf("synced=%q ready=%q", synced, ready)
		})

		waitFor(t, 5*time.Minute, "web deployment scaled to 3", func() (bool, string) {
			ready := kubectlMaybeGet("deployment", "-n", validNamespace, "web", "-o", `jsonpath={.status.readyReplicas}`)
			if ready == "3" {
				return true, "deployment/web has 3 ready replicas"
			}
			return false, fmt.Sprintf("readyReplicas=%q", ready)
		})

		elapsed := time.Since(start)
		if elapsed > 5*time.Minute {
			t.Fatalf("component enable path exceeded SLA: %s", elapsed)
		}

		webImage := kubectlGet(t, "deployment", "-n", validNamespace, "web", "-o", `jsonpath={.spec.template.spec.containers[0].image}`)
		if !strings.Contains(webImage, "1.27.4") {
			t.Fatalf("deployment/web image tag override not applied: %q", webImage)
		}

		webReplicas := kubectlGet(t, "deployment", "-n", validNamespace, "web", "-o", `jsonpath={.status.readyReplicas}`)
		if webReplicas != "3" {
			t.Fatalf("expected 3 ready web replicas, got %q", webReplicas)
		}

		dbReady := kubectlGet(t, "xpostgresqlinstance.database.idp.platform.io", "-n", validNamespace, "-o", `jsonpath={.items[0].status.conditions[?(@.type=="Ready")].status}`)
		if dbReady != "True" {
			t.Fatalf("database XR is not Ready: %q", dbReady)
		}

		if kubectlMaybeGet("secret", "-n", validNamespace, "database-db-credentials", "-o", "name") == "" {
			t.Fatal("database credentials secret was not created")
		}

		configValues := kubectlMaybeGet("configmap", "-n", validNamespace, "-o", `jsonpath={range .items[*].data[*]}{@}{"\n"}{end}`)
		if strings.Contains(strings.ToLower(configValues), "password") {
			t.Fatalf("configmap data unexpectedly contains password-like literal:\n%s", configValues)
		}

		envValues := kubectlMaybeGet("deployment", "-n", validNamespace, "-o", `jsonpath={range .items[*].spec.template.spec.containers[*].env[*]}{.value}{"\n"}{end}`)
		if strings.Contains(strings.ToLower(envValues), "password") {
			t.Fatalf("deployment env contains password-like literal:\n%s", envValues)
		}

		if kubectlMaybeGet("application", "-n", "argocd", validApp, "-o", "name") == "" {
			t.Fatalf("expected generated Application %q to exist", validApp)
		}
	})

	t.Run("AC-013-replicas-zero", func(t *testing.T) {
		repo.writeFile(t, validPath, componentsManifest(validName, 0, true))
		repo.commitAndPush(t, "Scale components acceptance manifest to zero "+validName)

		waitFor(t, 3*time.Minute, "deployment scaled to zero", func() (bool, string) {
			replicas := kubectlMaybeGet("deployment", "-n", validNamespace, "web", "-o", `jsonpath={.spec.replicas}`)
			ready := kubectlMaybeGet("deployment", "-n", validNamespace, "web", "-o", `jsonpath={.status.readyReplicas}`)
			if replicas == "0" && ready == "" {
				return true, "deployment exists with zero ready replicas"
			}
			return false, fmt.Sprintf("spec.replicas=%q readyReplicas=%q", replicas, ready)
		})

		pods := kubectlMaybeGet("pod", "-n", validNamespace, "-l", "app.kubernetes.io/instance=web", "-o", `jsonpath={.items[*].metadata.name}`)
		if strings.TrimSpace(pods) != "" {
			t.Fatalf("expected zero web pods when replicas=0, got %q", pods)
		}
	})

	t.Run("AC-016-disable-database", func(t *testing.T) {
		start := time.Now()
		repo.writeFile(t, validPath, componentsManifest(validName, 0, false))
		repo.commitAndPush(t, "Disable database component "+validName)

		waitFor(t, 3*time.Minute, "database resources removed", func() (bool, string) {
			deploy := kubectlMaybeGet("deployment", "-n", validNamespace, "database", "-o", "name")
			secret := kubectlMaybeGet("secret", "-n", validNamespace, "database-db-credentials", "-o", "name")
			pvc := kubectlMaybeGet("persistentvolumeclaim", "-n", validNamespace, "database-data", "-o", "name")
			if deploy == "" && secret == "" && pvc == "" {
				return true, "database runtime resources removed"
			}
			return false, fmt.Sprintf("deploy=%q secret=%q pvc=%q", deploy, secret, pvc)
		})

		if time.Since(start) > 3*time.Minute {
			t.Fatalf("database disable exceeded SLA: %s", time.Since(start))
		}
	})

	t.Run("AC-011-invalid-replicas-range", func(t *testing.T) {
		repo.writeFile(t, invalidPath, componentsManifest(invalidName, 51, false))
		repo.commitAndPush(t, "Add invalid replicas manifest "+invalidName)

		waitFor(t, 3*time.Minute, "invalid replicas rejected", func() (bool, string) {
			sync := kubectlMaybeGet("application", "-n", "argocd", invalidApp, "-o", `jsonpath={.status.sync.status}`)
			message := kubectlMaybeGet("application", "-n", "argocd", invalidApp, "-o", `jsonpath={.status.operationState.syncResult.resources[0].message}`)
			if sync == "OutOfSync" && strings.Contains(message, "should be less than or equal to 50") {
				return true, "application reports schema range validation failure"
			}
			return false, fmt.Sprintf("sync=%q message=%q", sync, message)
		})

		if kubectlMaybeGet("environment", "-n", controlNamespace, invalidName, "-o", "name") != "" {
			t.Fatalf("invalid replicas manifest unexpectedly created Environment claim")
		}
		if kubectlMaybeGet("namespace", invalidNamespace, "-o", "name") != "" {
			t.Fatalf("invalid replicas manifest unexpectedly created workload namespace")
		}
	})
}

func componentsManifest(name string, replicas int, databaseEnabled bool) string {
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
      replicas: %d
      imageRepository: docker.io/library/nginx
      imageTag: 1.27.4
    - name: database
      type: postgresql
      enabled: %t
`, name, testTeam, replicas, databaseEnabled)
}
