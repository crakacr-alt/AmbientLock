package app

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/crakacr-alt/AmbientLock/internal/model"
)

func TestLearnAppliesAmbientIgnore(t *testing.T) {
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

	tempDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.WriteFile(".ambientignore", []byte("read /etc/hostname\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"learn", "--lock", "ambient.lock", "--", catPath, "/etc/hostname"},
		&stdout,
		&stderr,
	)
	if code != ExitOK {
		t.Fatalf("learn failed: code=%d stderr=%s", code, stderr.String())
	}

	lock, err := model.Load("ambient.lock")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range lock.Filesystem.Reads {
		if path == "/etc/hostname" {
			t.Fatalf("ignored path is still present in lock: %#v", lock.Filesystem.Reads)
		}
	}
}

func TestDiffWritesJSONReport(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only integration test")
	}
	if _, err := exec.LookPath("strace"); err != nil {
		t.Skip("strace is not installed")
	}
	truePath, err := exec.LookPath("true")
	if err != nil {
		t.Skip("true is not installed")
	}

	tempDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"learn", "--no-ignore", "--lock", "baseline.lock", "--", truePath},
		&stdout,
		&stderr,
	)
	if code != ExitOK {
		t.Fatalf("learn failed: code=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(
		[]string{
			"diff",
			"--no-ignore",
			"--lock", "baseline.lock",
			"--json-report", "report.json",
			"--", truePath,
		},
		&stdout,
		&stderr,
	)
	if code != ExitOK && code != ExitDiffChanged {
		t.Fatalf("diff failed: code=%d stderr=%s", code, stderr.String())
	}

	payload, err := os.ReadFile(filepath.Join(tempDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}

	var report map[string]any
	if err := json.Unmarshal(payload, &report); err != nil {
		t.Fatal(err)
	}
	if _, ok := report["changed"]; !ok {
		t.Fatalf("JSON report has no changed field: %s", payload)
	}
	if _, ok := report["new_capabilities"]; !ok {
		t.Fatalf("JSON report has no new_capabilities field: %s", payload)
	}
}

func TestExplicitMissingIgnoreFileFailsBeforeTrace(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(
		[]string{"learn", "--ignore", "missing.ambientignore", "--", "anything"},
		&stdout,
		&stderr,
	)

	if code != ExitInternalError {
		t.Fatalf("unexpected code: %d", code)
	}
}
