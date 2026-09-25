package contract

import (
	"fmt"
	"sort"

	"github.com/crakacr-alt/AmbientLock/internal/model"
)

// ChangeSet is a human-readable difference between a saved contract and a new run.
// "Added" capabilities are the important part for CI enforcement; "Removed" items
// are still shown because they often explain why a dependency disappeared.
type ChangeSet struct {
	AddedExecutables   []string `json:"added_executables"`
	RemovedExecutables []string `json:"removed_executables"`
	AddedReads         []string `json:"added_reads"`
	RemovedReads       []string `json:"removed_reads"`
	AddedWrites        []string `json:"added_writes"`
	RemovedWrites      []string `json:"removed_writes"`
	AddedNetwork       []model.NetworkEndpoint `json:"added_network"`
	RemovedNetwork     []model.NetworkEndpoint `json:"removed_network"`
	AddedEnvNames      []string `json:"added_env_names"`
	RemovedEnvNames    []string `json:"removed_env_names"`
}

func (c ChangeSet) HasChanges() bool {
	return len(c.AddedExecutables)+len(c.RemovedExecutables)+
		len(c.AddedReads)+len(c.RemovedReads)+
		len(c.AddedWrites)+len(c.RemovedWrites)+
		len(c.AddedNetwork)+len(c.RemovedNetwork)+
		len(c.AddedEnvNames)+len(c.RemovedEnvNames) > 0
}

// HasNewCapabilities is intentionally narrower than HasChanges. Enforce mode
// should fail when the program gains a capability, not when an old dependency
// disappears.
func (c ChangeSet) HasNewCapabilities() bool {
	// Exposed ENV names are intentionally not strict by default in v0.1.
	// execve tells us that a variable was passed to the process, not that the
	// application actually read it. The CLI can opt into strict ENV handling.
	return len(c.AddedExecutables)+len(c.AddedReads)+len(c.AddedWrites)+len(c.AddedNetwork) > 0
}

func Diff(oldLock, newLock model.LockFile) ChangeSet {
	oldLock.Normalize()
	newLock.Normalize()

	addedExec, removedExec := diffStrings(oldLock.Executables, newLock.Executables)
	addedReads, removedReads := diffStrings(oldLock.Filesystem.Reads, newLock.Filesystem.Reads)
	addedWrites, removedWrites := diffStrings(oldLock.Filesystem.Writes, newLock.Filesystem.Writes)
	addedEnv, removedEnv := diffStrings(oldLock.Environment.ExposedNames, newLock.Environment.ExposedNames)
	addedNet, removedNet := diffNetwork(oldLock.Network, newLock.Network)

	return ChangeSet{
		AddedExecutables:   addedExec,
		RemovedExecutables: removedExec,
		AddedReads:         addedReads,
		RemovedReads:       removedReads,
		AddedWrites:        addedWrites,
		RemovedWrites:      removedWrites,
		AddedNetwork:       addedNet,
		RemovedNetwork:     removedNet,
		AddedEnvNames:      addedEnv,
		RemovedEnvNames:    removedEnv,
	}
}

func diffStrings(oldValues, newValues []string) (added, removed []string) {
	oldSet := toSet(oldValues)
	newSet := toSet(newValues)
	for value := range newSet {
		if _, ok := oldSet[value]; !ok {
			added = append(added, value)
		}
	}
	for value := range oldSet {
		if _, ok := newSet[value]; !ok {
			removed = append(removed, value)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func toSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func diffNetwork(oldValues, newValues []model.NetworkEndpoint) (added, removed []model.NetworkEndpoint) {
	oldSet := make(map[string]model.NetworkEndpoint, len(oldValues))
	newSet := make(map[string]model.NetworkEndpoint, len(newValues))
	for _, value := range oldValues {
		oldSet[networkKey(value)] = value
	}
	for _, value := range newValues {
		newSet[networkKey(value)] = value
	}
	for key, value := range newSet {
		if _, ok := oldSet[key]; !ok {
			added = append(added, value)
		}
	}
	for key, value := range oldSet {
		if _, ok := newSet[key]; !ok {
			removed = append(removed, value)
		}
	}
	sort.Slice(added, func(i, j int) bool { return networkKey(added[i]) < networkKey(added[j]) })
	sort.Slice(removed, func(i, j int) bool { return networkKey(removed[i]) < networkKey(removed[j]) })
	return added, removed
}

func networkKey(endpoint model.NetworkEndpoint) string {
	return fmt.Sprintf("%s|%s|%d", endpoint.Family, endpoint.Address, endpoint.Port)
}
