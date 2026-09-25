package contract

import (
	"fmt"
	"io"

	"github.com/crakacr-alt/AmbientLock/internal/model"
)

// Print writes a compact Git-friendly change report.
func Print(w io.Writer, changes ChangeSet) {
	printStrings(w, "+ executable", changes.AddedExecutables)
	printStrings(w, "- executable", changes.RemovedExecutables)
	printStrings(w, "+ read", changes.AddedReads)
	printStrings(w, "- read", changes.RemovedReads)
	printStrings(w, "+ write", changes.AddedWrites)
	printStrings(w, "- write", changes.RemovedWrites)
	printNetwork(w, "+ network", changes.AddedNetwork)
	printNetwork(w, "- network", changes.RemovedNetwork)
	printStrings(w, "+ env", changes.AddedEnvNames)
	printStrings(w, "- env", changes.RemovedEnvNames)
}

func printStrings(w io.Writer, prefix string, values []string) {
	for _, value := range values {
		fmt.Fprintf(w, "%s: %s\n", prefix, value)
	}
}

func printNetwork(w io.Writer, prefix string, values []model.NetworkEndpoint) {
	for _, value := range values {
		if value.Port > 0 {
			fmt.Fprintf(w, "%s: %s %s:%d\n", prefix, value.Family, value.Address, value.Port)
		} else {
			fmt.Fprintf(w, "%s: %s %s\n", prefix, value.Family, value.Address)
		}
	}
}
