package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const CurrentSchema = 1

type NetworkEndpoint struct {
	Family  string `json:"family"`
	Address string `json:"address"`
	Port    int    `json:"port,omitempty"`
}

type FileAccess struct {
	Reads  []string `json:"reads"`
	Writes []string `json:"writes"`
}

type Environment struct {
	ExposedNames []string `json:"exposed_names"`
}

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type LockFile struct {
	Schema      int               `json:"schema"`
	Tool        string            `json:"tool"`
	ToolVersion string            `json:"tool_version"`
	Command     []string          `json:"command"`
	Platform    Platform          `json:"platform"`
	Executables []string          `json:"executables"`
	Filesystem  FileAccess        `json:"filesystem"`
	Network     []NetworkEndpoint `json:"network"`
	Environment Environment       `json:"environment"`
}

func NewLock(version string, command []string, platform Platform) LockFile {
	return LockFile{
		Schema: CurrentSchema, Tool: "ambientlock", ToolVersion: version,
		Command: append([]string(nil), command...), Platform: platform,
	}
}

func (l *LockFile) Normalize() {
	l.Executables = uniqueSorted(l.Executables)
	l.Filesystem.Reads = uniqueSorted(l.Filesystem.Reads)
	l.Filesystem.Writes = uniqueSorted(l.Filesystem.Writes)
	l.Environment.ExposedNames = uniqueSorted(l.Environment.ExposedNames)
	seen := make(map[string]NetworkEndpoint)
	for _, endpoint := range l.Network {
		key := fmt.Sprintf("%s|%s|%d", endpoint.Family, endpoint.Address, endpoint.Port)
		seen[key] = endpoint
	}
	l.Network = l.Network[:0]
	for _, endpoint := range seen {
		l.Network = append(l.Network, endpoint)
	}
	sort.Slice(l.Network, func(i, j int) bool {
		if l.Network[i].Family != l.Network[j].Family { return l.Network[i].Family < l.Network[j].Family }
		if l.Network[i].Address != l.Network[j].Address { return l.Network[i].Address < l.Network[j].Address }
		return l.Network[i].Port < l.Network[j].Port
	})
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" { seen[value] = struct{}{} }
	}
	result := make([]string, 0, len(seen))
	for value := range seen { result = append(result, value) }
	sort.Strings(result)
	return result
}

func Save(path string, lock LockFile) error {
	lock.Normalize()
	payload, err := json.MarshalIndent(lock, "", "  ")
	if err != nil { return fmt.Errorf("marshal lock file: %w", err) }
	payload = append(payload, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil { return fmt.Errorf("create lock directory: %w", err) }
	tmp, err := os.CreateTemp(dir, ".ambient-lock-*")
	if err != nil { return fmt.Errorf("create temporary lock file: %w", err) }
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary lock file: %w", err)
	}
	if err := tmp.Close(); err != nil { return fmt.Errorf("close temporary lock file: %w", err) }
	if err := os.Rename(tmpName, path); err != nil { return fmt.Errorf("replace lock file: %w", err) }
	return nil
}

func Load(path string) (LockFile, error) {
	payload, err := os.ReadFile(path)
	if err != nil { return LockFile{}, fmt.Errorf("read lock file: %w", err) }
	var lock LockFile
	if err := json.Unmarshal(payload, &lock); err != nil { return LockFile{}, fmt.Errorf("parse lock file: %w", err) }
	if lock.Schema != CurrentSchema {
		return LockFile{}, fmt.Errorf("unsupported lock schema %d (supported: %d)", lock.Schema, CurrentSchema)
	}
	lock.Normalize()
	return lock, nil
}
