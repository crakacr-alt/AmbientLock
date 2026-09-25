package app

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/crakacr-alt/AmbientLock/internal/contract"
	"github.com/crakacr-alt/AmbientLock/internal/model"
	"github.com/crakacr-alt/AmbientLock/internal/trace"
)

// Version is kept in one place so the CLI, lock file, and releases agree.
const Version = "0.1.0"

// Exit codes used by AmbientLock. Child-process exit codes are preserved when
// possible; these values are reserved for AmbientLock's own decisions.
const (
	ExitOK            = 0
	ExitUsage         = 2
	ExitDiffChanged   = 3
	ExitNewCapability = 4
	ExitInternalError = 10
)

// Run is the testable CLI entry point. main() only forwards os.Args and streams.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return ExitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return ExitOK
	case "version", "--version":
		fmt.Fprintf(stdout, "AmbientLock v%s\n", Version)
		return ExitOK
	case "learn":
		return runLearn(args[1:], stdout, stderr)
	case "diff":
		return runDiff(args[1:], stdout, stderr, false)
	case "enforce":
		return runDiff(args[1:], stdout, stderr, true)
	case "show":
		return runShow(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printHelp(stderr)
		return ExitUsage
	}
}

func runLearn(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("learn", flag.ContinueOnError)
	flags.SetOutput(stderr)
	lockPath := flags.String("lock", "ambient.lock", "path to the lock file")
	if err := flags.Parse(args); err != nil {
		return ExitUsage
	}
	command := flags.Args()
	if len(command) == 0 {
		fmt.Fprintln(stderr, "learn requires a command after --")
		return ExitUsage
	}

	lock, exitCode, err := observe(command, stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "ambient:", err)
		return ExitInternalError
	}
	if err := model.Save(*lockPath, lock); err != nil {
		fmt.Fprintln(stderr, "ambient:", err)
		return ExitInternalError
	}

	fmt.Fprintf(stdout, "\nAmbientLock learned contract -> %s\n", *lockPath)
	printSummary(stdout, lock)
	if exitCode != 0 {
		fmt.Fprintf(stderr, "traced command exited with code %d; contract was still saved\n", exitCode)
		return exitCode
	}
	return ExitOK
}

func runDiff(args []string, stdout, stderr io.Writer, enforce bool) int {
	name := "diff"
	if enforce {
		name = "enforce"
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	lockPath := flags.String("lock", "ambient.lock", "path to the lock file")
	strictEnv := flags.Bool("strict-env", false, "treat newly exposed ENV names as enforcement failures")
	if err := flags.Parse(args); err != nil {
		return ExitUsage
	}
	command := flags.Args()
	if len(command) == 0 {
		fmt.Fprintf(stderr, "%s requires a command after --\n", name)
		return ExitUsage
	}

	baseline, err := model.Load(*lockPath)
	if err != nil {
		fmt.Fprintln(stderr, "ambient:", err)
		return ExitInternalError
	}
	current, exitCode, err := observe(command, stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "ambient:", err)
		return ExitInternalError
	}
	changes := contract.Diff(baseline, current)

	fmt.Fprintln(stdout, "\nAmbient contract diff:")
	if !changes.HasChanges() {
		fmt.Fprintln(stdout, "no changes")
	} else {
		contract.Print(stdout, changes)
	}

	if exitCode != 0 {
		fmt.Fprintf(stderr, "traced command exited with code %d\n", exitCode)
		return exitCode
	}
	if enforce {
		newCapabilities := changes.HasNewCapabilities()
		if *strictEnv && len(changes.AddedEnvNames) > 0 {
			newCapabilities = true
		}
		if newCapabilities {
			fmt.Fprintln(stderr, "enforce failed: new ambient capabilities detected")
			return ExitNewCapability
		}
		fmt.Fprintln(stdout, "enforce passed: no new capabilities")
		return ExitOK
	}
	if changes.HasChanges() {
		return ExitDiffChanged
	}
	return ExitOK
}

func runShow(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	lockPath := flags.String("lock", "ambient.lock", "path to the lock file")
	if err := flags.Parse(args); err != nil {
		return ExitUsage
	}
	if len(flags.Args()) != 0 {
		fmt.Fprintln(stderr, "show does not accept a command")
		return ExitUsage
	}
	lock, err := model.Load(*lockPath)
	if err != nil {
		fmt.Fprintln(stderr, "ambient:", err)
		return ExitInternalError
	}
	printSummary(stdout, lock)
	return ExitOK
}

func observe(command []string, stdout, stderr io.Writer) (model.LockFile, int, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return model.LockFile{}, 0, fmt.Errorf("get working directory: %w", err)
	}

	runner := trace.NewRunner()
	runner.Stdout = stdout
	runner.Stderr = stderr
	result, err := runner.Run(cwd, command)
	if err != nil {
		return model.LockFile{}, 0, err
	}

	lock := model.NewLock(Version, sanitizeCommand(command), model.Platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	})
	lock.Executables = result.Observation.Executables
	lock.Filesystem.Reads = result.Observation.Reads
	lock.Filesystem.Writes = result.Observation.Writes
	lock.Network = result.Observation.Network
	lock.Environment.ExposedNames = result.Observation.EnvNames
	lock.Normalize()
	return lock, result.ExitCode, nil
}

func sanitizeCommand(command []string) []string {
	result := append([]string(nil), command...)
	redactNext := false

	for i, arg := range result {
		if redactNext {
			result[i] = "<redacted>"
			redactNext = false
			continue
		}

		if !strings.HasPrefix(arg, "--") {
			continue
		}

		name, _, hasValue := strings.Cut(arg, "=")
		if !isSensitiveFlag(name) {
			continue
		}

		if hasValue {
			result[i] = name + "=<redacted>"
		} else {
			redactNext = true
		}
	}

	return result
}

func isSensitiveFlag(flagName string) bool {
	name := strings.TrimLeft(strings.ToLower(flagName), "-")
	for _, marker := range []string{
		"password", "passwd", "token", "secret", "api-key", "apikey",
		"authorization", "credential", "client-secret", "access-key",
	} {
		if name == marker || strings.HasSuffix(name, "-"+marker) {
			return true
		}
	}
	return false
}

func printSummary(w io.Writer, lock model.LockFile) {
	fmt.Fprintf(w, "schema: %d\n", lock.Schema)
	fmt.Fprintf(w, "command: %s\n", strings.Join(lock.Command, " "))
	fmt.Fprintf(w, "platform: %s/%s\n", lock.Platform.OS, lock.Platform.Arch)
	fmt.Fprintf(w, "executables: %d\n", len(lock.Executables))
	fmt.Fprintf(w, "file reads: %d\n", len(lock.Filesystem.Reads))
	fmt.Fprintf(w, "file writes: %d\n", len(lock.Filesystem.Writes))
	fmt.Fprintf(w, "network endpoints: %d\n", len(lock.Network))
	fmt.Fprintf(w, "environment names exposed: %d\n", len(lock.Environment.ExposedNames))
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `AmbientLock — lock-файл для скрытых зависимостей программы.

Usage:
  ambient learn [--lock ambient.lock] -- <command> [args...]
  ambient diff [--lock ambient.lock] [--strict-env] -- <command> [args...]
  ambient enforce [--lock ambient.lock] [--strict-env] -- <command> [args...]
  ambient show [--lock ambient.lock]
  ambient version

Commands:
  learn    выполнить команду под наблюдением и создать ambient.lock
  diff     повторить запуск и показать изменения контракта
  enforce  вернуть ошибку, если появились НОВЫЕ capabilities
           ENV names по умолчанию только показываются; --strict-env делает их строгими
  show     показать краткую сводку lock-файла

Important:
  v0.1 supports Linux + strace. "enforce" is detective: it fails after the
  traced run if new capabilities were observed; it does not block syscalls yet.`)
}
