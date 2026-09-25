package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Report struct {
	Changed         bool      `json:"changed"`
	NewCapabilities bool      `json:"new_capabilities"`
	StrictEnv       bool      `json:"strict_env"`
	Changes         ChangeSet `json:"changes"`
}

func BuildReport(changes ChangeSet, strictEnv bool) Report {
	newCapabilities := changes.HasNewCapabilities()
	if strictEnv && len(changes.AddedEnvNames) > 0 {
		newCapabilities = true
	}

	return Report{
		Changed:         changes.HasChanges(),
		NewCapabilities: newCapabilities,
		StrictEnv:       strictEnv,
		Changes:         changes,
	}
}

func SaveReport(path string, report Report) error {
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON report: %w", err)
	}
	payload = append(payload, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create JSON report directory: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write JSON report: %w", err)
	}
	return nil
}
