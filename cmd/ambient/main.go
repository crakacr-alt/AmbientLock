package main

import (
	"os"

	"github.com/crakacr-alt/AmbientLock/internal/app"
)

// main intentionally stays tiny. All logic lives in internal packages where it
// can be unit-tested without spawning a second CLI process.
func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
