package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildReportStrictEnv(t *testing.T) {
	changes := ChangeSet{AddedEnvNames: []string{"NEW_TOKEN"}}

	normal := BuildReport(changes, false)
	if normal.NewCapabilities {
		t.Fatal("ENV must stay informational without strict mode")
	}

	strict := BuildReport(changes, true)
	if !strict.NewCapabilities {
		t.Fatal("strict ENV must count as new capability")
	}
}

func TestSaveReportProducesJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifacts", "report.json")
	report := BuildReport(ChangeSet{AddedReads: []string{"./config.yaml"}}, false)

	if err := SaveReport(path, report); err != nil {
		t.Fatal(err)
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["changed"] != true {
		t.Fatalf("unexpected report: %s", payload)
	}
}
