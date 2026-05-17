package mcp

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// MetadataVerificationNote explains the scope of MCP discovery checks.
	MetadataVerificationNote = "metadata health only; end-to-end MCP execution is not verified"
)

// ServerDescriptor describes a configured MCP server.
type ServerDescriptor struct {
	ID            string `json:"id"`
	Source        string `json:"source"`
	Enabled       bool   `json:"enabled"`
	ToolCount     int    `json:"tool_count"`
	ResourceCount int    `json:"resource_count"`
	Status        string `json:"status,omitempty"`
}

// InvocationRequest describes one MCP tool execution request.
type InvocationRequest struct {
	ServerID  string         `json:"serverId"`
	ToolName  string         `json:"toolName"`
	Arguments map[string]any `json:"arguments"`
}

// InvocationResult describes the normalized result returned by a synthetic MCP tool.
type InvocationResult struct {
	ServerID string         `json:"serverId"`
	ToolName string         `json:"toolName"`
	Content  string         `json:"content"`
	Data     map[string]any `json:"data,omitempty"`
}

// ToolExecutor executes an MCP tool call.
type ToolExecutor func(arguments map[string]any) (InvocationResult, error)

// Manager stores MCP server metadata.
type Manager struct {
	servers    []ServerDescriptor
	executors  map[string]map[string]ToolExecutor
	serverByID map[string]ServerDescriptor
}

// NewManager builds a new Manager.
func NewManager(servers []ServerDescriptor) *Manager {
	cloned := make([]ServerDescriptor, len(servers))
	for i, server := range servers {
		cloned[i] = NormalizeServerDescriptor(server)
	}
	items := make(map[string]ServerDescriptor, len(cloned))
	for _, item := range cloned {
		items[item.ID] = item
	}
	return &Manager{
		servers:    cloned,
		executors:  make(map[string]map[string]ToolExecutor),
		serverByID: items,
	}
}

// List returns configured MCP servers sorted by ID.
func (m *Manager) List() []ServerDescriptor {
	items := make([]ServerDescriptor, len(m.servers))
	copy(items, m.servers)
	for i, item := range items {
		items[i] = NormalizeServerDescriptor(item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

// NormalizeServerDescriptor returns a descriptor with stable metadata health semantics.
func NormalizeServerDescriptor(server ServerDescriptor) ServerDescriptor {
	server.Status = NormalizeServerStatus(server)
	return server
}

// NormalizeServerStatus derives the stable metadata health label for a server descriptor.
func NormalizeServerStatus(server ServerDescriptor) string {
	status := strings.ToLower(strings.TrimSpace(server.Status))
	switch status {
	case "enabled", "disabled", "degraded", "unreachable":
		if !server.Enabled {
			return "disabled"
		}
		return status
	}
	if !server.Enabled {
		return "disabled"
	}
	if server.ToolCount == 0 && server.ResourceCount == 0 {
		return "degraded"
	}
	return "enabled"
}

// MetadataHealthSummary formats a server using the shared metadata-health wording.
func MetadataHealthSummary(server ServerDescriptor) string {
	server = NormalizeServerDescriptor(server)
	return fmt.Sprintf("%s (%s, metadata health: %s)", server.ID, metadataEnabledLabel(server), server.Status)
}

func metadataEnabledLabel(server ServerDescriptor) string {
	if !server.Enabled {
		return "disabled"
	}
	return "enabled"
}

// RegisterExecutor stores an executable synthetic MCP tool under the given server and tool name.
func (m *Manager) RegisterExecutor(serverID string, toolName string, exec ToolExecutor) {
	if m == nil || exec == nil {
		return
	}
	serverID = strings.TrimSpace(serverID)
	toolName = strings.TrimSpace(toolName)
	if serverID == "" || toolName == "" {
		return
	}
	if _, ok := m.executors[serverID]; !ok {
		m.executors[serverID] = make(map[string]ToolExecutor)
	}
	m.executors[serverID][toolName] = exec
}

// Invoke executes a registered synthetic MCP tool.
func (m *Manager) Invoke(request InvocationRequest) (InvocationResult, error) {
	if m == nil {
		return InvocationResult{}, fmt.Errorf("mcp manager unavailable")
	}
	request.ServerID = strings.TrimSpace(request.ServerID)
	request.ToolName = strings.TrimSpace(request.ToolName)
	if request.ServerID == "" {
		return InvocationResult{}, fmt.Errorf("mcp server id is required")
	}
	if request.ToolName == "" {
		return InvocationResult{}, fmt.Errorf("mcp tool name is required")
	}
	server, ok := m.serverByID[request.ServerID]
	if !ok {
		return InvocationResult{}, fmt.Errorf("mcp server %s not found", request.ServerID)
	}
	if !server.Enabled {
		return InvocationResult{}, fmt.Errorf("mcp server %s is disabled", request.ServerID)
	}
	tools, ok := m.executors[request.ServerID]
	if !ok {
		return InvocationResult{}, fmt.Errorf("mcp server %s has no executable tools", request.ServerID)
	}
	exec, ok := tools[request.ToolName]
	if !ok {
		return InvocationResult{}, fmt.Errorf("mcp tool %s/%s not found", request.ServerID, request.ToolName)
	}
	result, err := exec(request.Arguments)
	if err != nil {
		return InvocationResult{}, err
	}
	if strings.TrimSpace(result.ServerID) == "" {
		result.ServerID = request.ServerID
	}
	if strings.TrimSpace(result.ToolName) == "" {
		result.ToolName = request.ToolName
	}
	return result, nil
}
