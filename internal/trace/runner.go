package trace

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

// RunResult contains both the observed capabilities and the child's exit code.
type RunResult struct {
	Observation Observation
	ExitCode    int
}

// Runner executes a command under strace. Keeping tracing behind this small
// type makes it possible to add eBPF, macOS, or Windows backends later without
// changing the lock-file model.
type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func NewRunner() *Runner {
	return &Runner{Stdout: os.Stdout, Stderr: os.Stderr}
}

func (r *Runner) Run(cwd string, command []string) (RunResult, error) {
	if runtime.GOOS != "linux" {
		return RunResult{}, fmt.Errorf("v0.1 tracing supports Linux only (current OS: %s)", runtime.GOOS)
	}
	if len(command) == 0 {
		return RunResult{}, errors.New("command is empty")
	}
	stracePath, err := exec.LookPath("strace")
	if err != nil {
		return RunResult{}, errors.New("strace not found; install it first (for Debian/Ubuntu: sudo apt install strace)")
	}

	tempDir, err := os.MkdirTemp("", "ambientlock-trace-")
	if err != nil {
		return RunResult{}, fmt.Errorf("create trace directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	prefix := filepath.Join(tempDir, "trace")

	args := []string{
		"-ff", "-qq", "-v", "-s", "4096", "-yy",
		"-e", "trace=execve,open,openat,creat,connect",
		"-o", prefix,
		"--",
	}
	args = append(args, command...)

	cmd := exec.Command(stracePath, args...)
	cmd.Dir = cwd
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr

	exitCode := 0
	runErr := cmd.Run()
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return RunResult{}, fmt.Errorf("start strace: %w", runErr)
		}
	}

	traceFiles, err := filepath.Glob(prefix + "*")
	if err != nil {
		return RunResult{}, fmt.Errorf("list trace files: %w", err)
	}
	if len(traceFiles) == 0 {
		return RunResult{}, fmt.Errorf("strace produced no trace files; command may not exist")
	}
	sort.Strings(traceFiles)

	parser := NewParser(cwd)
	observations := make([]Observation, 0, len(traceFiles))
	for _, path := range traceFiles {
		file, err := os.Open(path)
		if err != nil {
			return RunResult{}, fmt.Errorf("open trace file %s: %w", path, err)
		}
		observation, parseErr := parser.Parse(file)
		closeErr := file.Close()
		if parseErr != nil {
			return RunResult{}, parseErr
		}
		if closeErr != nil {
			return RunResult{}, fmt.Errorf("close trace file %s: %w", path, closeErr)
		}
		observations = append(observations, observation)
	}

	return RunResult{Observation: Merge(observations...), ExitCode: exitCode}, nil
}
