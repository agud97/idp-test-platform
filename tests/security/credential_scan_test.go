package security

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var literalSecretPattern = regexp.MustCompile(`(?i)^\s*(password|secret|token|privateKey)\s*:\s*['"]?[^'"#\s][^#]*$`)

func TestEnvironmentYAMLContainsNoLiteralCredentials(t *testing.T) {
	root := repoRoot(t)
	envDir := filepath.Join(root, "environments")

	var findings []string
	err := filepath.Walk(envDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if literalSecretPattern.MatchString(line) {
				findings = append(findings, path+":"+strconv.Itoa(lineNo)+": "+line)
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatalf("walk environments/: %v", err)
	}

	if len(findings) > 0 {
		t.Fatalf("found literal credential values in environments/:\n%s", strings.Join(findings, "\n"))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
