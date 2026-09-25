package trace

import (
	"strings"
	"testing"
)

func TestParserExtractsCapabilitiesWithoutEnvironmentValues(t *testing.T) {
	input := strings.Join([]string{
		`101 execve("/usr/bin/python3", ["python3", "app.py"], ["PATH=/usr/bin", "API_TOKEN=super-secret", "PWD=/work"]) = 0`,
		`101 openat(AT_FDCWD, "config.yaml", O_RDONLY|O_CLOEXEC) = 3</work/config.yaml>`,
		`101 openat(AT_FDCWD, "/var/tmp/output.txt", O_WRONLY|O_CREAT|O_TRUNC, 0666) = 4</var/tmp/output.txt>`,
		`101 connect(5<TCP:[123]>, {sa_family=AF_INET, sin_port=htons(443), sin_addr=inet_addr("203.0.113.10")}, 16) = 0`,
	}, "\n")

	got, err := NewParser("/work").Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Executables) != 1 || got.Executables[0] != "/usr/bin/python3" {
		t.Fatalf("unexpected executables: %#v", got.Executables)
	}
	if len(got.Reads) != 1 || got.Reads[0] != "./config.yaml" {
		t.Fatalf("unexpected reads: %#v", got.Reads)
	}
	if len(got.Writes) != 1 || got.Writes[0] != "/var/tmp/output.txt" {
		t.Fatalf("unexpected writes: %#v", got.Writes)
	}
	if len(got.Network) != 1 || got.Network[0].Address != "203.0.113.10" || got.Network[0].Port != 443 {
		t.Fatalf("unexpected network: %#v", got.Network)
	}

	joined := strings.Join(got.EnvNames, ",")
	if joined != "PATH,API_TOKEN" && joined != "API_TOKEN,PATH" {
		t.Fatalf("unexpected env names: %#v", got.EnvNames)
	}
	if strings.Contains(joined, "super-secret") {
		t.Fatal("environment value leaked into observation")
	}
}

func TestParserIgnoresFailedAndKernelPseudoFileAccess(t *testing.T) {
	input := strings.Join([]string{
		`openat(AT_FDCWD, "/does/not/exist", O_RDONLY) = -1 ENOENT (No such file or directory)`,
		`openat(AT_FDCWD, "/proc/self/status", O_RDONLY) = 3`,
	}, "\n")

	got, err := NewParser("/work").Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Reads) != 0 {
		t.Fatalf("expected no recorded reads, got %#v", got.Reads)
	}
}

func TestCreatIsWriteOnly(t *testing.T) {
	got, err := NewParser("/work").Parse(strings.NewReader(`creat("result.txt", 0666) = 3</work/result.txt>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Reads) != 0 {
		t.Fatalf("creat must not be classified as read: %#v", got.Reads)
	}
	if len(got.Writes) != 1 || got.Writes[0] != "./result.txt" {
		t.Fatalf("unexpected writes: %#v", got.Writes)
	}
}

func TestOpenatUsesKernelResolvedPath(t *testing.T) {
	input := `openat(AT_FDCWD</work/sub>, "config.yaml", O_RDONLY|O_CLOEXEC) = 3</work/sub/config.yaml>`

	got, err := NewParser("/work").Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Reads) != 1 || got.Reads[0] != "./sub/config.yaml" {
		t.Fatalf("expected kernel-resolved path, got %#v", got.Reads)
	}
}

func TestOpenatDirectoryFDUsesKernelResolvedPath(t *testing.T) {
	input := `openat(3</opt/data>, "item.txt", O_RDONLY) = 4</opt/data/item.txt>`

	got, err := NewParser("/work").Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Reads) != 1 || got.Reads[0] != "/opt/data/item.txt" {
		t.Fatalf("expected directory-fd resolved path, got %#v", got.Reads)
	}
}
