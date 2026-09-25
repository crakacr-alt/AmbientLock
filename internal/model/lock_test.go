package model

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveLoadNormalizesSets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ambient.lock")
	lock := NewLock("test", []string{"echo", "ok"}, Platform{OS: "linux", Arch: "amd64"})
	lock.Executables = []string{"/bin/z", "/bin/a", "/bin/a"}
	lock.Filesystem.Reads = []string{"/b", "/a", "/a"}
	lock.Environment.ExposedNames = []string{"TOKEN", "PATH", "TOKEN"}

	if err := Save(path, lock); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(loaded.Executables, []string{"/bin/a", "/bin/z"}) {
		t.Fatalf("unexpected executables: %#v", loaded.Executables)
	}
	if !reflect.DeepEqual(loaded.Environment.ExposedNames, []string{"PATH", "TOKEN"}) {
		t.Fatalf("unexpected environment names: %#v", loaded.Environment.ExposedNames)
	}
}
