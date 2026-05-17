package mcp

import "sort"

// ServerDescriptor describes a configured MCP server.
type ServerDescriptor struct {
	ID            string `json:"id"`
	Source        string `json:"source"`
	Enabled       bool   `json:"enabled"`
	ToolCount     int    `json:"tool_count"`
	ResourceCount int    `json:"resource_count"`
	Status        string `json:"status,omitempty"`
}

// Manager stores MCP server metadata.
type Manager struct {
	servers []ServerDescriptor
}

// NewManager builds a new Manager.
func NewManager(servers []ServerDescriptor) *Manager {
	cloned := make([]ServerDescriptor, len(servers))
	copy(cloned, servers)
	return &Manager{servers: cloned}
}

// List returns configured MCP servers sorted by ID.
func (m *Manager) List() []ServerDescriptor {
	items := make([]ServerDescriptor, len(m.servers))
	copy(items, m.servers)
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}
