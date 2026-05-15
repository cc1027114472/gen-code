package runtime

import (
	"sort"

	"llmtrace/internal/core/state"
)

const defaultVersion = "0.1.0"

// NewDefaultService creates the minimal phase-one runtime used by server, CLI, and desktop.
func NewDefaultService() *Service {
	discovered := discoverSiblingRuntimeContent(workspaceRoot())
	return newServiceFromDiscoveryWithStore(discovered, nil)
}

// NewDefaultServiceWithStateStore creates the shared runtime with an explicit state store.
func NewDefaultServiceWithStateStore(store *state.Store) *Service {
	discovered := discoverSiblingRuntimeContent(workspaceRoot())
	return newServiceFromDiscoveryWithStore(discovered, store)
}

// SkillGroups returns the concrete skill names grouped for CLI inspection.
func SkillGroups() map[string][]string {
	resolver := newSkillResolver(discoverSiblingRuntimeContent(workspaceRoot()))
	groups := map[string][]string{"common": resolver.Common()}
	if group, ok := resolver.Resolve("codex"); ok {
		groups["codex"] = append([]string(nil), group.Skills...)
	}
	if group, ok := resolver.Resolve("cc"); ok {
		groups["cc"] = append([]string(nil), group.Skills...)
	}

	for name, items := range groups {
		cloned := append([]string(nil), items...)
		sort.Strings(cloned)
		groups[name] = cloned
	}
	return groups
}
