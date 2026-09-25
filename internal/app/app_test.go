package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"version"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("unexpected exit code: %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), Version) {
		t.Fatalf("version output does not contain %s: %s", Version, stdout.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"wat"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestSanitizeCommandRedactsCommonSecretFlags(t *testing.T) {
	got := sanitizeCommand([]string{
		"tool",
		"--token", "very-secret",
		"--api-key=abc123",
		"--output", "report.json",
	})
	want := []string{
		"tool",
		"--token", "<redacted>",
		"--api-key=<redacted>",
		"--output", "report.json",
	}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected sanitized command: %#v", got)
	}
}

func TestSanitizeCommandDoesNotRedactOrdinaryArguments(t *testing.T) {
	got := sanitizeCommand([]string{"tool", "--profile", "secret-lab", "file-token.txt"})
	want := []string{"tool", "--profile", "secret-lab", "file-token.txt"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("ordinary arguments changed: %#v", got)
	}
}
