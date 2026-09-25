package trace

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/crakacr-alt/AmbientLock/internal/model"
)

var (
	quotedStringRE = regexp.MustCompile(`"(?:\\.|[^"\\])*"`)
	resultRE       = regexp.MustCompile(`\)\s+=\s+(-?\d+)`)
	ipv4RE         = regexp.MustCompile(`sin_port=htons\((\d+)\).*sin_addr=inet_addr\("([^"]+)"\)`)
	ipv6RE         = regexp.MustCompile(`sin6_port=htons\((\d+)\).*inet_pton\(AF_INET6,\s*"([^"]+)"`)
	unixRE         = regexp.MustCompile(`sun_path="([^"]+)"`)
	openedPathRE   = regexp.MustCompile(`\)\s+=\s+\d+<([^>]+)>`)
)

// Observation is the raw set of capabilities seen during one traced run.
type Observation struct {
	Executables []string
	Reads       []string
	Writes      []string
	Network     []model.NetworkEndpoint
	EnvNames    []string
}

// Parser turns human-readable strace output into a small, versioned contract.
// We intentionally support only syscalls used by AmbientLock v0.1; unsupported
// lines are ignored instead of guessed.
type Parser struct {
	cwd string
}

func NewParser(cwd string) *Parser { return &Parser{cwd: cwd} }

// Parse reads one strace output stream. It is safe to call it repeatedly for
// different -ff trace files and merge the resulting observations.
func (p *Parser) Parse(reader io.Reader) (Observation, error) {
	var result Observation
	scanner := bufio.NewScanner(reader)
	// execve environment arrays can exceed Scanner's default 64 KiB buffer.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := stripPIDPrefix(strings.TrimSpace(scanner.Text()))
		switch {
		case strings.HasPrefix(line, "execve("):
			p.parseExecve(line, &result)
		case strings.HasPrefix(line, "openat("), strings.HasPrefix(line, "open("), strings.HasPrefix(line, "creat("):
			p.parseOpen(line, &result)
		case strings.HasPrefix(line, "connect("):
			p.parseConnect(line, &result)
		}
	}
	if err := scanner.Err(); err != nil {
		return Observation{}, fmt.Errorf("scan strace output: %w", err)
	}
	return result, nil
}

func stripPIDPrefix(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 2 { return line }
	if _, err := strconv.Atoi(fields[0]); err == nil {
		return strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
	}
	return line
}

func (p *Parser) parseExecve(line string, result *Observation) {
	if !syscallSucceeded(line) { return }
	quoted := quotedStrings(line)
	if len(quoted) == 0 { return }
	result.Executables = append(result.Executables, p.normalizePath(quoted[0]))

	// strace -v prints envp. Keep names only: values may contain secrets.
	marker := strings.Index(line, "], [")
	if marker < 0 { return }
	envPart := line[marker+4:]
	for _, item := range quotedStrings(envPart) {
		name, _, ok := strings.Cut(item, "=")
		if ok && validEnvName(name) && !ignoredEnvName(name) {
			result.EnvNames = append(result.EnvNames, name)
		}
	}
}

func (p *Parser) parseOpen(line string, result *Observation) {
	if !syscallSucceeded(line) { return }
	quoted := quotedStrings(line)
	if len(quoted) == 0 { return }

	// -yy appends the kernel-resolved path to a successful returned fd. Prefer
	// it because it remains correct after chdir and with directory-fd openat.
	pathText := quoted[0]
	if match := openedPathRE.FindStringSubmatch(line); len(match) == 2 && filepath.IsAbs(match[1]) {
		pathText = match[1]
	}
	path := p.normalizePath(pathText)
	if ignoredPath(path) { return }

	isCreat := strings.HasPrefix(line, "creat(")
	write := strings.Contains(line, "O_WRONLY") || strings.Contains(line, "O_RDWR") ||
		strings.Contains(line, "O_CREAT") || strings.Contains(line, "O_TRUNC") ||
		strings.Contains(line, "O_APPEND") || isCreat
	read := !isCreat && (!strings.Contains(line, "O_WRONLY") || strings.Contains(line, "O_RDWR"))

	if read { result.Reads = append(result.Reads, path) }
	if write { result.Writes = append(result.Writes, path) }
}

func (p *Parser) parseConnect(line string, result *Observation) {
	// EINPROGRESS is normal for a non-blocking connect attempt.
	if !syscallSucceeded(line) && !strings.Contains(line, "EINPROGRESS") { return }
	if match := ipv4RE.FindStringSubmatch(line); len(match) == 3 {
		port, _ := strconv.Atoi(match[1])
		result.Network = append(result.Network, model.NetworkEndpoint{Family: "ipv4", Address: match[2], Port: port})
		return
	}
	if match := ipv6RE.FindStringSubmatch(line); len(match) == 3 {
		port, _ := strconv.Atoi(match[1])
		result.Network = append(result.Network, model.NetworkEndpoint{Family: "ipv6", Address: match[2], Port: port})
		return
	}
	if match := unixRE.FindStringSubmatch(line); len(match) == 2 {
		result.Network = append(result.Network, model.NetworkEndpoint{Family: "unix", Address: match[1]})
	}
}

func syscallSucceeded(line string) bool {
	match := resultRE.FindStringSubmatch(line)
	if len(match) != 2 { return false }
	value, err := strconv.Atoi(match[1])
	return err == nil && value >= 0
}

func quotedStrings(text string) []string {
	matches := quotedStringRE.FindAllString(text, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		value, err := strconv.Unquote(match)
		if err == nil { result = append(result, value) }
	}
	return result
}

func (p *Parser) normalizePath(path string) string {
	if path == "" { return path }
	if !filepath.IsAbs(path) { path = filepath.Join(p.cwd, path) }
	path = filepath.Clean(path)
	relative, err := filepath.Rel(p.cwd, path)
	if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "./" + filepath.ToSlash(relative)
	}
	return path
}

func ignoredPath(path string) bool {
	for _, prefix := range []string{"/proc/", "/sys/", "/dev/"} {
		if strings.HasPrefix(path, prefix) { return true }
	}
	return strings.Contains(path, "/ambientlock-trace-")
}

func validEnvName(name string) bool {
	if name == "" { return false }
	for i, r := range name {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || (i > 0 && r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func ignoredEnvName(name string) bool {
	switch name {
	case "_", "PWD", "OLDPWD", "SHLVL":
		return true
	default:
		return false
	}
}

// Merge combines observations from several strace -ff process files.
func Merge(items ...Observation) Observation {
	var merged Observation
	for _, item := range items {
		merged.Executables = append(merged.Executables, item.Executables...)
		merged.Reads = append(merged.Reads, item.Reads...)
		merged.Writes = append(merged.Writes, item.Writes...)
		merged.Network = append(merged.Network, item.Network...)
		merged.EnvNames = append(merged.EnvNames, item.EnvNames...)
	}
	return merged
}
