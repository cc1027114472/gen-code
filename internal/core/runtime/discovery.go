package runtime

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"unicode"

	"llmtrace/internal/core/mcp"
	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/provider"
	"llmtrace/internal/core/session"
	"llmtrace/internal/core/skill"
	"llmtrace/internal/core/state"
	"llmtrace/internal/core/tool"
)

type discoverySet struct {
	skills []skill.Descriptor
	tools  []tool.Descriptor
	mcp    []mcp.ServerDescriptor
}

func discoverSiblingRuntimeContent(workspaceRoot string) discoverySet {
	parentRoot := filepath.Dir(workspaceRoot)

	codexSkillsRoot := filepath.Join(parentRoot, "codex", ".codex", "skills")
	ccSkillsRoots := []string{
		filepath.Join(parentRoot, "CC ibwhale", ".claude", "skills"),
		filepath.Join(parentRoot, "CC ibwhale", "ibwhale", ".claude", "skills"),
	}
	ccToolsRoot := filepath.Join(parentRoot, "CC ibwhale", "ibwhale", "tools")
	ccNodeModulesRoot := filepath.Join(parentRoot, "CC ibwhale", "node_modules")

	discoveredSkills := []skill.Descriptor{
		{ID: "common.browser", Group: skill.Common, Name: "Browser", Description: "Common browser automation skill"},
	}
	discoveredSkills = append(discoveredSkills, discoverSkills(codexSkillsRoot, skill.Codex)...)
	for _, root := range ccSkillsRoots {
		discoveredSkills = append(discoveredSkills, discoverSkills(root, skill.CC)...)
	}

	discoveredTools := []tool.Descriptor{
		{
			ID:                 "bridge.check",
			Name:               "Bridge Check",
			Description:        "Verify the desktop/runtime bridge",
			InputSchemaSummary: "No input",
			PermissionMode:     policy.AskUser,
			Source:             "runtime",
			Kind:               "bridge",
			ReadOnly:           true,
			Executable:         false,
		},
		{
			ID:                 "skills.list",
			Name:               "Skills List",
			Description:        "List runtime-visible skills",
			InputSchemaSummary: "Optional skill group",
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "runtime.discovery",
			ReadOnly:           true,
			Executable:         false,
		},
		{
			ID:                 "mcp.servers.list",
			Name:               "MCP Servers List",
			Description:        "List configured MCP servers",
			InputSchemaSummary: "No input",
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "runtime.discovery",
			ReadOnly:           true,
			Executable:         false,
		},
		{
			ID:                 "workspace.read_file",
			Name:               "Workspace Read File",
			Description:        "Read a file under the current workspace root",
			InputSchemaSummary: `{"path":"relative/or/absolute path within workspace"}`,
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "workspace.read_file",
			ReadOnly:           true,
			Executable:         true,
		},
		{
			ID:                 "workspace.list_files",
			Name:               "Workspace List Files",
			Description:        "List files under a workspace-relative directory",
			InputSchemaSummary: `{"path":"optional relative directory within workspace"}`,
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "workspace.list_files",
			ReadOnly:           true,
			Executable:         true,
		},
		{
			ID:                 "workspace.search_text",
			Name:               "Workspace Search Text",
			Description:        "Search workspace files for a text pattern",
			InputSchemaSummary: `{"query":"text to search","path":"optional relative directory within workspace"}`,
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "workspace.search_text",
			ReadOnly:           true,
			Executable:         true,
		},
		{
			ID:                 "thread.message.append",
			Name:               "Thread Message Append",
			Description:        "Append a private message to the current thread context",
			InputSchemaSummary: `{"role":"user|assistant|system","content":"message body"}`,
			PermissionMode:     policy.AskUser,
			Source:             "runtime",
			Kind:               "thread.message.append",
			ReadOnly:           false,
			Executable:         true,
		},
	}
	discoveredTools = append(discoveredTools, discoverTools(ccToolsRoot)...)

	discoveredMCP := discoverMCPServers(ccNodeModulesRoot)
	if len(discoveredMCP) == 0 {
		discoveredMCP = append(discoveredMCP, mcp.ServerDescriptor{
			ID:            "local-files",
			Source:        "builtin",
			Enabled:       true,
			ToolCount:     1,
			ResourceCount: 1,
		})
	}

	return discoverySet{
		skills: dedupeSkillDescriptors(discoveredSkills),
		tools:  dedupeToolDescriptors(discoveredTools),
		mcp:    dedupeMCPDescriptors(discoveredMCP),
	}
}

func discoverSkills(root string, group skill.Group) []skill.Descriptor {
	ids := discoverIDs(root, func(entry os.DirEntry) (string, bool) {
		switch {
		case entry.IsDir():
			return entry.Name(), true
		case strings.EqualFold(filepath.Ext(entry.Name()), ".md"):
			return strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), true
		default:
			return "", false
		}
	})

	items := make([]skill.Descriptor, 0, len(ids))
	for _, id := range ids {
		items = append(items, skill.Descriptor{
			ID:          id,
			Group:       group,
			Name:        humanizeName(id),
			Description: groupDescription(group),
		})
	}
	return items
}

func discoverTools(root string) []tool.Descriptor {
	ids := discoverIDs(root, func(entry os.DirEntry) (string, bool) {
		switch {
		case entry.IsDir():
			return entry.Name(), true
		case entry.Type().IsRegular():
			return strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), true
		default:
			return "", false
		}
	})

	items := make([]tool.Descriptor, 0, len(ids))
	for _, id := range ids {
		items = append(items, tool.Descriptor{
			ID:                 id,
			Name:               humanizeName(id),
			Description:        "Discovered CC project tool",
			InputSchemaSummary: "Project-defined tool input",
			PermissionMode:     policy.AskUser,
			Source:             "cc",
			Kind:               "external",
			ReadOnly:           false,
			Executable:         false,
		})
	}
	return items
}

func discoverMCPServers(root string) []mcp.ServerDescriptor {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	items := make([]mcp.ServerDescriptor, 0)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.Contains(strings.ToLower(name), "modelcontextprotocol") {
			continue
		}
		items = append(items, mcp.ServerDescriptor{
			ID:            name,
			Source:        "node_modules",
			Enabled:       true,
			ToolCount:     0,
			ResourceCount: 0,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func dedupeSkillDescriptors(items []skill.Descriptor) []skill.Descriptor {
	seen := map[string]skill.Descriptor{}
	for _, item := range items {
		key := string(item.Group) + ":" + item.ID
		seen[key] = item
	}
	result := make([]skill.Descriptor, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Group == result[j].Group {
			return result[i].ID < result[j].ID
		}
		return result[i].Group < result[j].Group
	})
	return result
}

func dedupeToolDescriptors(items []tool.Descriptor) []tool.Descriptor {
	seen := map[string]tool.Descriptor{}
	for _, item := range items {
		seen[item.ID] = item
	}
	result := make([]tool.Descriptor, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func dedupeMCPDescriptors(items []mcp.ServerDescriptor) []mcp.ServerDescriptor {
	seen := map[string]mcp.ServerDescriptor{}
	for _, item := range items {
		seen[item.ID] = item
	}
	result := make([]mcp.ServerDescriptor, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func humanizeName(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func groupDescription(group skill.Group) string {
	switch group {
	case skill.Codex:
		return "Discovered from codex skills"
	case skill.CC:
		return "Discovered from cc skills"
	default:
		return "Shared runtime skill"
	}
}

func discoverIDs(root string, include func(os.DirEntry) (string, bool)) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{}, len(entries))
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		id, ok := include(entry)
		if !ok || id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		items = append(items, id)
	}
	sort.Strings(items)
	return items
}

func workspaceRoot() string {
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}

func newServiceFromDiscoveryWithStore(discovered discoverySet, explicitStore *state.Store, providers *provider.Registry) *Service {
	registry := tool.NewRegistry()
	for _, item := range discovered.tools {
		registry.Register(item)
	}
	skills := skill.NewManager(discovered.skills)
	mcpManager := mcp.NewManager(discovered.mcp)
	projectRoot := workspaceRoot()
	store := explicitStore
	if store == nil {
		opened, err := state.Open(projectRoot)
		if err != nil {
			sessions := session.NewRegistry(projectRoot)
			return NewService(defaultVersion, skill.Common, policy.DefaultMode(), projectRoot, registry, skills, mcpManager, providers, sessions)
		}
		store = opened
	}
	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	if err != nil {
		sessions = session.NewRegistry(projectRoot)
	}
	return NewService(defaultVersion, skill.Common, policy.DefaultMode(), projectRoot, registry, skills, mcpManager, providers, sessions)
}

func newSkillResolver(discovered discoverySet) *skill.Resolver {
	common := make([]string, 0)
	grouped := map[string][]string{
		"codex": {},
		"cc":    {},
	}

	for _, item := range discovered.skills {
		switch item.Group {
		case skill.Common:
			common = append(common, item.ID)
		case skill.Codex:
			grouped["codex"] = append(grouped["codex"], item.ID)
		case skill.CC:
			grouped["cc"] = append(grouped["cc"], item.ID)
		}
	}

	return skill.NewResolver(common, grouped)
}
