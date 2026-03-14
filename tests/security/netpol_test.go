package security

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const defaultKubeconfig = "/root/codex/kubeconfig_6144665"

var requiredPolicies = []string{
	"allow-dns-egress",
	"allow-intra-namespace",
	"deny-cross-namespace-ingress",
}

func TestEnvironmentNamespacesHaveBaselineNetworkPolicies(t *testing.T) {
	ensureKubeconfig(t)

	namespaces := strings.Fields(runKubectl(t, "get", "ns", "-o", `jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}`))
	var envNamespaces []string
	for _, namespace := range namespaces {
		if strings.HasPrefix(namespace, "env-") {
			envNamespaces = append(envNamespaces, namespace)
		}
	}
	if len(envNamespaces) == 0 {
		t.Skip("no environment namespaces found")
	}

	for _, namespace := range envNamespaces {
		namespace := namespace
		t.Run(namespace, func(t *testing.T) {
			policies := strings.Fields(runKubectl(t, "get", "networkpolicy", "-n", namespace, "-o", `jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}`))
			for _, required := range requiredPolicies {
				if !contains(policies, required) {
					t.Fatalf("namespace %q missing required NetworkPolicy %q; got %v", namespace, required, policies)
				}
			}
		})
	}
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

func runKubectl(t *testing.T, args ...string) string {
	t.Helper()

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		cmd := exec.CommandContext(ctx, "kubectl", append([]string{"--request-timeout=15s"}, args...)...)
		cmd.Env = os.Environ()
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		cancel()
		if err == nil {
			return stdout.String()
		}
		lastErr = err
		time.Sleep(2 * time.Second)
		if ctx.Err() == nil && stderr.Len() > 0 {
			lastErr = execError(stderr.String())
		}
	}

	t.Fatalf("kubectl %s failed after retries: %v", strings.Join(args, " "), lastErr)
	return ""
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type execError string

func (e execError) Error() string { return string(e) }
