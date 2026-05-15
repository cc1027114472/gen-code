package skill

import "sort"

// ResolvedGroup represents the outcome of resolving a skill group with common reuse.
type ResolvedGroup struct {
	Name       string
	UsesCommon bool
	Skills     []string
}

// Resolver resolves group names into concrete skill lists.
type Resolver struct {
	common []string
	groups map[string][]string
}

// NewResolver creates a resolver with shared common skills and named groups.
func NewResolver(common []string, groups map[string][]string) *Resolver {
	copiedCommon := append([]string(nil), common...)
	copiedGroups := make(map[string][]string, len(groups))
	for name, skills := range groups {
		copiedGroups[name] = append([]string(nil), skills...)
	}

	return &Resolver{common: copiedCommon, groups: copiedGroups}
}

// Resolve returns a group with common skills reused unless disabled.
func (r *Resolver) Resolve(name string) (ResolvedGroup, bool) {
	skills, ok := r.groups[name]
	if !ok {
		return ResolvedGroup{}, false
	}

	merged := make([]string, 0, len(r.common)+len(skills))
	merged = append(merged, r.common...)
	merged = append(merged, skills...)

	return ResolvedGroup{
		Name:       name,
		UsesCommon: len(r.common) > 0,
		Skills:     dedupeSorted(merged),
	}, true
}

// Common returns a copy of the shared skills.
func (r *Resolver) Common() []string {
	return append([]string(nil), r.common...)
}

func dedupeSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
