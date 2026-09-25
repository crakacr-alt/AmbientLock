package policy

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/crakacr-alt/AmbientLock/internal/model"
)

func TestRulesApply(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ambientignore")
	content := `# temporary files
./.cache/**
write ./tmp/**
env CI_*
network ipv4:127.0.0.1:*
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if rules.Len() != 4 {
		t.Fatalf("unexpected rule count: %d", rules.Len())
	}

	lock := model.LockFile{
		Filesystem: model.FileAccess{
			Reads:  []string{"./.cache/index", "./config.yaml"},
			Writes: []string{"./tmp/out.txt", "./result.json"},
		},
		Network: []model.NetworkEndpoint{
			{Family: "ipv4", Address: "127.0.0.1", Port: 5432},
			{Family: "ipv4", Address: "203.0.113.10", Port: 443},
		},
		Environment: model.Environment{ExposedNames: []string{"CI_JOB_ID", "HOME"}},
	}

	rules.Apply(&lock)

	if !reflect.DeepEqual(lock.Filesystem.Reads, []string{"./config.yaml"}) {
		t.Fatalf("unexpected reads: %#v", lock.Filesystem.Reads)
	}
	if !reflect.DeepEqual(lock.Filesystem.Writes, []string{"./result.json"}) {
		t.Fatalf("unexpected writes: %#v", lock.Filesystem.Writes)
	}
	if !reflect.DeepEqual(lock.Environment.ExposedNames, []string{"HOME"}) {
		t.Fatalf("unexpected env: %#v", lock.Environment.ExposedNames)
	}
	if len(lock.Network) != 1 || lock.Network[0].Address != "203.0.113.10" {
		t.Fatalf("unexpected network: %#v", lock.Network)
	}
}

func TestDoubleStarMatchesDirectories(t *testing.T) {
	re, err := compileGlob("./cache/**")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"./cache/a", "./cache/a/b/c"} {
		if !re.MatchString(value) {
			t.Fatalf("pattern did not match %q", value)
		}
	}
}

func TestBarePatternWithSpacesIsAllowed(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ambientignore")
	if err := os.WriteFile(path, []byte("wat ./tmp/**\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A line without a known kind is treated as a normal file pattern.
	// This protects paths that happen to contain spaces.
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
}

func TestTypedRuleNeedsPattern(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ambientignore")
	if err := os.WriteFile(path, []byte("read\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected empty typed rule to fail")
	}
}
