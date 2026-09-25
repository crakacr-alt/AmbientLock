package trace

import (
	"bytes"
	"os/exec"
	"runtime"
	"testing"
)

func TestRunnerSeesExecutableAndFileRead(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only integration test")
	}
	if _, err := exec.LookPath("strace"); err != nil {
		t.Skip("strace is not installed")
	}
	catPath, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat is not installed")
	}

	var stdout, stderr bytes.Buffer
	runner := NewRunner()
	runner.Stdout = &stdout
	runner.Stderr = &stderr
	result, err := runner.Run(t.TempDir(), []string{catPath, "/etc/hostname"})
	if err != nil {
		t.Fatalf("runner error: %v, stderr=%s", err, stderr.String())
	}
	if result.ExitCode != 0 {
		t.Fatalf("unexpected child exit code: %d", result.ExitCode)
	}

	if !contains(result.Observation.Executables, catPath) {
		t.Fatalf("executable %q not observed: %#v", catPath, result.Observation.Executables)
	}
	if !contains(result.Observation.Reads, "/etc/hostname") {
		t.Fatalf("/etc/hostname not observed: %#v", result.Observation.Reads)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
