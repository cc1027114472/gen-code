package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"time"
)

const (
	// MetadataVerificationNote explains the scope of MCP discovery checks.
	MetadataVerificationNote = "metadata health only; end-to-end MCP execution is not verified"
	// ExecutionVerificationNote explains the scope of the executable baseline.
	ExecutionVerificationNote = "multi-server stdio external execution baseline verified"
)

var (
	ErrServerNotFound = errors.New("mcp server not found")
	ErrToolNotFound   = errors.New("mcp tool not found")
)

// ServerDescriptor describes a configured MCP server.
type ServerDescriptor struct {
	ID               string            `json:"id"`
	Source           string            `json:"source"`
	Enabled          bool              `json:"enabled"`
	ToolCount        int               `json:"tool_count"`
	ResourceCount    int               `json:"resource_count"`
	Status           string            `json:"status,omitempty"`
	Command          []string          `json:"-"`
	Env              map[string]string `json:"-"`
	Tools            []string          `json:"-"`
	Transport        string            `json:"-"`
	ExecutionTier    string            `json:"-"`
	ExecutionSummary string            `json:"-"`
}

// InvokeRequest is the minimal MCP execution payload.
type InvokeRequest struct {
	ServerID  string         `json:"serverId"`
	ToolName  string         `json:"toolName"`
	Arguments map[string]any `json:"arguments"`
}

// InvokeResult is the normalized execution result returned to runner/CLI.
type InvokeResult struct {
	ServerID      string         `json:"serverId"`
	ToolName      string         `json:"toolName"`
	Transport     string         `json:"transport"`
	Result        map[string]any `json:"result,omitempty"`
	ResultSummary string         `json:"resultSummary"`
}

type invokeEnvelope struct {
	ToolName  string         `json:"toolName"`
	Arguments map[string]any `json:"arguments"`
}

type invokeResponseEnvelope struct {
	OK      bool           `json:"ok"`
	Error   string         `json:"error,omitempty"`
	Summary string         `json:"summary,omitempty"`
	Result  map[string]any `json:"result,omitempty"`
}

// Manager stores MCP server metadata plus optional execution config.
type Manager struct {
	servers map[string]ServerDescriptor
}

// NewManager builds a new Manager.
func NewManager(servers []ServerDescriptor) *Manager {
	items := make(map[string]ServerDescriptor, len(servers))
	for _, server := range servers {
		normalized := NormalizeServerDescriptor(server)
		items[normalized.ID] = normalized
	}
	return &Manager{servers: items}
}

// List returns configured MCP servers sorted by ID.
func (m *Manager) List() []ServerDescriptor {
	if m == nil {
		return nil
	}
	items := make([]ServerDescriptor, 0, len(m.servers))
	for _, item := range m.servers {
		items = append(items, NormalizeServerDescriptor(item))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

// CanInvoke reports whether at least one configured server has an execution adapter.
func (m *Manager) CanInvoke() bool {
	if m == nil {
		return false
	}
	for _, item := range m.servers {
		if len(item.Command) > 0 {
			return true
		}
	}
	return false
}

// Invoke executes a minimal MCP tool call through the configured external adapter.
func (m *Manager) Invoke(ctx context.Context, request InvokeRequest) (InvokeResult, error) {
	if m == nil {
		return InvokeResult{}, ErrServerNotFound
	}
	server, ok := m.servers[strings.TrimSpace(request.ServerID)]
	if !ok {
		return InvokeResult{}, fmt.Errorf("%w: %s", ErrServerNotFound, strings.TrimSpace(request.ServerID))
	}
	if !containsTool(server.Tools, request.ToolName) {
		return InvokeResult{}, fmt.Errorf("%w: %s/%s", ErrToolNotFound, server.ID, request.ToolName)
	}
	if len(server.Command) == 0 {
		return InvokeResult{}, fmt.Errorf("mcp server %s is metadata-only", server.ID)
	}

	switch normalizedTransport(server.Transport) {
	case "stdio-fixture":
		return m.invokeFixtureServer(ctx, server, request)
	default:
		return m.invokeBridgeServer(ctx, server, request)
	}
}

func (m *Manager) invokeFixtureServer(ctx context.Context, server ServerDescriptor, request InvokeRequest) (InvokeResult, error) {
	payload, err := json.Marshal(invokeEnvelope{
		ToolName:  request.ToolName,
		Arguments: cloneArguments(request.Arguments),
	})
	if err != nil {
		return InvokeResult{}, err
	}

	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	if _, ok := callCtx.Deadline(); !ok {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, 10*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(callCtx, server.Command[0], server.Command[1:]...)
	cmd.Env = commandEnvironment(server.Env)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return InvokeResult{}, fmt.Errorf("external mcp server %s failed: %s", server.ID, detail)
	}

	var response invokeResponseEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return InvokeResult{}, fmt.Errorf("external mcp server %s returned invalid json: %w", server.ID, err)
	}
	if !response.OK {
		failure := strings.TrimSpace(response.Error)
		if failure == "" {
			failure = "unknown external mcp error"
		}
		return InvokeResult{}, fmt.Errorf("external mcp server %s failed: %s", server.ID, failure)
	}

	summary := strings.TrimSpace(response.Summary)
	if summary == "" {
		summary = fmt.Sprintf("mcp tool %s/%s executed", server.ID, request.ToolName)
	}

	return InvokeResult{
		ServerID:      server.ID,
		ToolName:      request.ToolName,
		Transport:     normalizedTransport(server.Transport),
		Result:        response.Result,
		ResultSummary: summary,
	}, nil
}

func (m *Manager) invokeBridgeServer(ctx context.Context, server ServerDescriptor, request InvokeRequest) (InvokeResult, error) {
	payload, err := json.Marshal(map[string]any{
		"serverId":   server.ID,
		"transport":  normalizedTransport(server.Transport),
		"command":    server.Command,
		"env":        cloneEnvironment(server.Env),
		"toolName":   request.ToolName,
		"arguments":  cloneArguments(request.Arguments),
		"summary":    fmt.Sprintf("mcp tool %s/%s executed", server.ID, request.ToolName),
		"serverHint": server.ExecutionSummary,
	})
	if err != nil {
		return InvokeResult{}, err
	}

	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	if _, ok := callCtx.Deadline(); !ok {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, 15*time.Second)
		defer cancel()
	}

	bridgeCommand := bridgeCommand()
	cmd := exec.CommandContext(callCtx, bridgeCommand[0], bridgeCommand[1:]...)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return InvokeResult{}, fmt.Errorf("external mcp server %s failed: %s", server.ID, detail)
	}

	var response invokeResponseEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return InvokeResult{}, fmt.Errorf("external mcp server %s returned invalid json: %w", server.ID, err)
	}
	if !response.OK {
		failure := strings.TrimSpace(response.Error)
		if failure == "" {
			failure = "unknown external mcp error"
		}
		return InvokeResult{}, fmt.Errorf("external mcp server %s failed: %s", server.ID, failure)
	}

	summary := strings.TrimSpace(response.Summary)
	if summary == "" {
		summary = fmt.Sprintf("mcp tool %s/%s executed", server.ID, request.ToolName)
	}

	return InvokeResult{
		ServerID:      server.ID,
		ToolName:      request.ToolName,
		Transport:     normalizedTransport(server.Transport),
		Result:        response.Result,
		ResultSummary: summary,
	}, nil
}

// NormalizeServerDescriptor returns a descriptor with stable metadata health semantics.
func NormalizeServerDescriptor(server ServerDescriptor) ServerDescriptor {
	server.Status = NormalizeServerStatus(server)
	server.Tools = dedupeTools(server.Tools)
	server.Transport = normalizedTransport(server.Transport)
	server.ExecutionTier = strings.TrimSpace(server.ExecutionTier)
	server.ExecutionSummary = strings.TrimSpace(server.ExecutionSummary)
	server.Env = cloneEnvironment(server.Env)
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

func dedupeTools(tools []string) []string {
	if len(tools) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	items := make([]string, 0, len(tools))
	for _, tool := range tools {
		trimmed := strings.TrimSpace(tool)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	sort.Strings(items)
	return items
}

func containsTool(tools []string, toolName string) bool {
	for _, item := range tools {
		if item == strings.TrimSpace(toolName) {
			return true
		}
	}
	return false
}

func cloneArguments(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneEnvironment(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func normalizedTransport(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "stdio-fixture"
	}
	return trimmed
}

func commandEnvironment(overrides map[string]string) []string {
	if len(overrides) == 0 {
		return nil
	}
	base := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		base[key] = value
	}
	for key, value := range overrides {
		base[key] = value
	}
	env := make([]string, 0, len(base))
	for key, value := range base {
		env = append(env, key+"="+value)
	}
	sort.Strings(env)
	return env
}

func bridgeCommand() []string {
	return []string{"node", bridgeScriptPath()}
}

func bridgeScriptPath() string {
	return filepath.Join(mcpWorkspaceRoot(), "scripts", "mcp_stdio_bridge.js")
}

func mcpWorkspaceRoot() string {
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
