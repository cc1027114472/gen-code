package tool

import (
	"sort"
	"sync"

	"llmtrace/internal/core/policy"
)

// Descriptor describes a tool exposed by the runtime.
type Descriptor struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	InputSchemaSummary string      `json:"input_schema_summary"`
	PermissionMode     policy.Mode `json:"permission_mode"`
	Source             string      `json:"source"`
}

// Registry stores tool descriptors for runtime discovery.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Descriptor
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Descriptor)}
}

// Register stores or replaces a tool descriptor.
func (r *Registry) Register(desc Descriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[desc.ID] = desc
}

// List returns all registered tools sorted by ID.
func (r *Registry) List() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Descriptor, 0, len(r.tools))
	for _, desc := range r.tools {
		items = append(items, desc)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}
