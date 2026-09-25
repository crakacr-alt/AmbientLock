package policy

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/crakacr-alt/AmbientLock/internal/model"
)

type rule struct {
	kind    string
	pattern string
	re      *regexp.Regexp
}

// Rules is the parsed .ambientignore file.
// A rule can target read, write, exec, env, network, or file (read+write).
type Rules struct {
	items []rule
}

// Load reads ignore rules. The file format is intentionally small:
//
//	./.cache/**          # same as "file ./.cache/**"
//	read /etc/ssl/**
//	write ./tmp/**
//	exec /usr/bin/helper
//	env CI_*
//	network ipv4:127.0.0.1:*
func Load(path string) (Rules, error) {
	file, err := os.Open(path)
	if err != nil {
		return Rules{}, err
	}
	defer file.Close()

	var rules Rules
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		kind, pattern := splitRule(line)
		switch kind {
		case "file", "read", "write", "exec", "env", "network":
		default:
			return Rules{}, fmt.Errorf("%s:%d: unknown ignore kind %q", path, lineNumber, kind)
		}

		re, err := compileGlob(pattern)
		if err != nil {
			return Rules{}, fmt.Errorf("%s:%d: %w", path, lineNumber, err)
		}
		rules.items = append(rules.items, rule{kind: kind, pattern: pattern, re: re})
	}

	if err := scanner.Err(); err != nil {
		return Rules{}, fmt.Errorf("read ignore file: %w", err)
	}
	return rules, nil
}

func splitRule(line string) (string, string) {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		switch fields[0] {
		case "file", "read", "write", "exec", "env", "network":
			return fields[0], strings.TrimSpace(line[len(fields[0]):])
		}
	}
	return "file", line
}

func compileGlob(pattern string) (*regexp.Regexp, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("empty ignore pattern")
	}

	var b strings.Builder
	b.WriteString("^")

	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]

		if ch == '*' {
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
			continue
		}
		if ch == '?' {
			b.WriteString("[^/]")
			continue
		}

		b.WriteString(regexp.QuoteMeta(string(ch)))
	}

	b.WriteString("$")
	return regexp.Compile(b.String())
}

func (r Rules) Len() int {
	return len(r.items)
}

// Apply removes ignored entries from a lock before it is stored or compared.
func (r Rules) Apply(lock *model.LockFile) {
	if lock == nil || len(r.items) == 0 {
		return
	}

	lock.Executables = filterStrings(lock.Executables, func(value string) bool {
		return !r.matches("exec", value)
	})
	lock.Filesystem.Reads = filterStrings(lock.Filesystem.Reads, func(value string) bool {
		return !r.matches("read", value)
	})
	lock.Filesystem.Writes = filterStrings(lock.Filesystem.Writes, func(value string) bool {
		return !r.matches("write", value)
	})
	lock.Environment.ExposedNames = filterStrings(lock.Environment.ExposedNames, func(value string) bool {
		return !r.matches("env", value)
	})

	filteredNetwork := lock.Network[:0]
	for _, endpoint := range lock.Network {
		if !r.matches("network", networkTarget(endpoint)) {
			filteredNetwork = append(filteredNetwork, endpoint)
		}
	}
	lock.Network = filteredNetwork
	lock.Normalize()
}

func (r Rules) matches(kind, value string) bool {
	for _, item := range r.items {
		if item.kind != kind && !(item.kind == "file" && (kind == "read" || kind == "write")) {
			continue
		}
		if item.re.MatchString(value) {
			return true
		}
	}
	return false
}

func networkTarget(endpoint model.NetworkEndpoint) string {
	if endpoint.Port > 0 {
		return endpoint.Family + ":" + endpoint.Address + ":" + strconv.Itoa(endpoint.Port)
	}
	return endpoint.Family + ":" + endpoint.Address
}

func filterStrings(values []string, keep func(string) bool) []string {
	result := values[:0]
	for _, value := range values {
		if keep(value) {
			result = append(result, value)
		}
	}
	return result
}
