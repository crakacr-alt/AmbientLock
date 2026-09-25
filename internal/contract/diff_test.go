package contract

import (
	"testing"

	"github.com/crakacr-alt/AmbientLock/internal/model"
)

func TestDiffDetectsNewCapabilities(t *testing.T) {
	oldLock := model.LockFile{
		Executables: []string{"/usr/bin/python3"},
		Filesystem:  model.FileAccess{Reads: []string{"./config.yaml"}},
	}
	newLock := oldLock
	newLock.Executables = []string{"/usr/bin/python3", "/usr/bin/curl"}
	newLock.Filesystem = model.FileAccess{
		Reads:  []string{"./config.yaml"},
		Writes: []string{"/tmp/result.txt"},
	}
	newLock.Network = []model.NetworkEndpoint{{Family: "ipv4", Address: "203.0.113.10", Port: 443}}

	changes := Diff(oldLock, newLock)
	if !changes.HasChanges() || !changes.HasNewCapabilities() {
		t.Fatal("expected new capabilities")
	}
	if len(changes.AddedExecutables) != 1 || changes.AddedExecutables[0] != "/usr/bin/curl" {
		t.Fatalf("unexpected executable diff: %#v", changes.AddedExecutables)
	}
}

func TestRemovingDependencyDoesNotCountAsNewCapability(t *testing.T) {
	oldLock := model.LockFile{Executables: []string{"/bin/a", "/bin/b"}}
	newLock := model.LockFile{Executables: []string{"/bin/a"}}
	changes := Diff(oldLock, newLock)
	if !changes.HasChanges() {
		t.Fatal("expected a removal")
	}
	if changes.HasNewCapabilities() {
		t.Fatal("removal must not count as new capability")
	}
}

func TestNewExposedEnvIsInformationalByDefault(t *testing.T) {
	oldLock := model.LockFile{Environment: model.Environment{ExposedNames: []string{"PATH"}}}
	newLock := model.LockFile{Environment: model.Environment{ExposedNames: []string{"PATH", "CI_JOB_TOKEN"}}}

	changes := Diff(oldLock, newLock)
	if !changes.HasChanges() {
		t.Fatal("expected ENV diff to be visible")
	}
	if changes.HasNewCapabilities() {
		t.Fatal("exposed ENV name must be informational by default")
	}
}
