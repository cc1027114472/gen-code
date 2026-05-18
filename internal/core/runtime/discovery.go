package runtime

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"unicode"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/mcp"
	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/provider"
	"llmtrace/internal/core/session"
	"llmtrace/internal/core/skill"
	"llmtrace/internal/core/state"
	"llmtrace/internal/core/tool"
	"llmtrace/pkg/skillaudit"
)

type discoverySet struct {
	skills []skill.Descriptor
	tools  []tool.Descriptor
	mcp    []mcp.ServerDescriptor
}

type SkillGovernanceSummary struct {
	Group               string
	ImplementedCount    int
	VerifiedCount       int
	LocalizationPending int
	CapabilityPending   int
}

var builtinBrowserToolDescriptors = []tool.Descriptor{
	{
		ID:                 "browser.state",
		Name:               "Browser State",
		Description:        "Inspect the current browser workspace state",
		InputSchemaSummary: "No input",
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.state",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.open",
		Name:               "Browser Open",
		Description:        "Open a new controlled local browser tab for a URL",
		InputSchemaSummary: `{"url":"http://127.0.0.1:3000/"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.open",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.navigate",
		Name:               "Browser Navigate",
		Description:        "Navigate an existing controlled local browser tab to a URL",
		InputSchemaSummary: `{"tabId":"tab-1","url":"http://127.0.0.1:3000/"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.navigate",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.back",
		Name:               "Browser Back",
		Description:        "Navigate the active browser tab backward",
		InputSchemaSummary: `{"tabId":"tab-1"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.back",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.forward",
		Name:               "Browser Forward",
		Description:        "Navigate the active browser tab forward",
		InputSchemaSummary: `{"tabId":"tab-1"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.forward",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.reload",
		Name:               "Browser Reload",
		Description:        "Reload the active browser tab",
		InputSchemaSummary: `{"tabId":"tab-1"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.reload",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.close_tab",
		Name:               "Browser Close Tab",
		Description:        "Close a browser tab",
		InputSchemaSummary: `{"tabId":"tab-1"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.close_tab",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.activate_tab",
		Name:               "Browser Activate Tab",
		Description:        "Activate a browser tab",
		InputSchemaSummary: `{"tabId":"tab-1"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.activate_tab",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.click",
		Name:               "Browser Click",
		Description:        "Click a selector inside a controlled local browser tab",
		InputSchemaSummary: `{"tabId":"tab-1","selector":"[data-testid='apply']"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.click",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.type",
		Name:               "Browser Type",
		Description:        "Type text into a selector inside a controlled local browser tab",
		InputSchemaSummary: `{"tabId":"tab-1","selector":"[data-testid='name']","text":"hello"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.type",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.extract",
		Name:               "Browser Extract",
		Description:        "Extract text from a selector inside a controlled local browser tab",
		InputSchemaSummary: `{"tabId":"tab-1","selector":"[data-testid='result']"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.extract",
		ReadOnly:           true,
		Executable:         true,
	},
	{
		ID:                 "browser.screenshot",
		Name:               "Browser Screenshot",
		Description:        "Capture a screenshot for a controlled local browser tab",
		InputSchemaSummary: `{"tabId":"tab-1"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.screenshot",
		ReadOnly:           true,
		Executable:         true,
	},
}

const (
	skillIsolationStatusIsolated = "isolated"
	skillIsolationStatusShared   = "shared-common"
	skillIsolationStatusBlocked  = "blocked"
)

func discoverSiblingRuntimeContent(workspaceRoot string) discoverySet {
	parentRoot := filepath.Dir(workspaceRoot)

	codexSkillsRoot := filepath.Join(workspaceRoot, "internal", "core", "skill", "catalog", "codex")
	ccSkillsRoots := []string{
		filepath.Join(workspaceRoot, "internal", "core", "skill", "catalog", "cc"),
	}
	ccToolsRoot := filepath.Join(parentRoot, "CC ibwhale", "ibwhale", "tools")
	ccNodeModulesRoot := filepath.Join(parentRoot, "CC ibwhale", "node_modules")

	discoveredSkills := []skill.Descriptor{
		{
			ID:                  "common.browser",
			Group:               skill.Common,
			Name:                "Browser",
			Description:         "Common browser automation skill",
			Source:              "common",
			VerificationStatus:  "implemented",
			LocalizationChecked: true,
			IsolationStatus:     skillIsolationStatusShared,
			CapabilityVerified:  false,
			CapabilitySummary:   "capability baseline not tracked for built-in shared common skill",
		},
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
			ID:                 "workspace.stat_file",
			Name:               "Workspace Stat File",
			Description:        "Inspect a workspace file or directory metadata",
			InputSchemaSummary: `{"path":"workspace-relative path"}`,
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "workspace.stat_file",
			ReadOnly:           true,
			Executable:         true,
		},
		{
			ID:                 "workspace.read_files_batch",
			Name:               "Workspace Read Files Batch",
			Description:        "Read multiple text files under the current workspace root",
			InputSchemaSummary: `{"paths":["a.txt","docs/b.md"]}`,
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "workspace.read_files_batch",
			ReadOnly:           true,
			Executable:         true,
		},
		{
			ID:                 "workspace.list_files_filtered",
			Name:               "Workspace List Files Filtered",
			Description:        "List workspace entries filtered by a glob pattern",
			InputSchemaSummary: `{"path":"optional relative directory","pattern":"*.go","includeDirs":false}`,
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "workspace.list_files_filtered",
			ReadOnly:           true,
			Executable:         true,
		},
		{
			ID:                 "workspace.search_text_detailed",
			Name:               "Workspace Search Text Detailed",
			Description:        "Search workspace text with file and line details",
			InputSchemaSummary: `{"query":"text to search","path":"optional relative directory within workspace","limit":20}`,
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "workspace.search_text_detailed",
			ReadOnly:           true,
			Executable:         true,
		},
		{
			ID:                 "workspace.apply_patch",
			Name:               "Workspace Apply Patch",
			Description:        "Apply an approved text patch inside the current workspace root",
			InputSchemaSummary: `{"path":"relative workspace path","patch":"*** Begin Patch..."}`,
			PermissionMode:     policy.AskUser,
			Source:             "runtime",
			Kind:               "workspace.apply_patch",
			ReadOnly:           false,
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
		{
			ID:                 "mcp.tool.invoke",
			Name:               "MCP Tool Invoke",
			Description:        "Invoke a runtime-configured MCP tool through the shared task runner",
			InputSchemaSummary: `{"serverId":"sdk-external-fixture","toolName":"echo","arguments":{"message":"hello"}}`,
			PermissionMode:     policy.ReadOnly,
			Source:             "runtime",
			Kind:               "mcp.tool.invoke",
			ReadOnly:           true,
			Executable:         true,
		},
	}
	discoveredTools = append(discoveredTools, builtinBrowserToolDescriptors...)
	discoveredTools = append(discoveredTools, discoverTools(ccToolsRoot)...)

	discoveredMCP := discoverMCPServers(ccNodeModulesRoot)
	discoveredMCP = append(discoveredMCP, builtinExternalMCPFixture())
	discoveredMCP = append(discoveredMCP, builtinSDKExternalMCPServer())
	discoveredMCP = append(discoveredMCP, builtinThirdPartyTimeMCPServer())
	if len(discoveredMCP) == 0 {
		discoveredMCP = append(discoveredMCP, mcp.NormalizeServerDescriptor(mcp.ServerDescriptor{
			ID:            "local-files",
			Source:        "builtin",
			Enabled:       true,
			ToolCount:     1,
			ResourceCount: 1,
			Status:        "enabled",
		}))
	}

	return discoverySet{
		skills: dedupeSkillDescriptors(discoveredSkills),
		tools:  dedupeToolDescriptors(discoveredTools),
		mcp:    dedupeMCPDescriptors(discoveredMCP),
	}
}

func builtinExternalMCPFixture() mcp.ServerDescriptor {
	return mcp.NormalizeServerDescriptor(mcp.ServerDescriptor{
		ID:               "external-fixture",
		Source:           "fixture",
		Enabled:          true,
		ToolCount:        3,
		ResourceCount:    0,
		Status:           "enabled",
		Command:          mcpFixtureCommand(),
		Tools:            []string{"echo", "sum", "fail"},
		Transport:        "stdio-fixture",
		ExecutionTier:    "regression",
		ExecutionSummary: "fixture regression lane",
	})
}

func builtinSDKExternalMCPServer() mcp.ServerDescriptor {
	return mcp.NormalizeServerDescriptor(mcp.ServerDescriptor{
		ID:               "sdk-external-fixture",
		Source:           "sdk",
		Enabled:          true,
		ToolCount:        2,
		ResourceCount:    0,
		Status:           "enabled",
		Command:          mcpSDKServerCommand(),
		Tools:            []string{"echo", "sum"},
		Transport:        "stdio-sdk",
		ExecutionTier:    "canonical-verified",
		ExecutionSummary: "official SDK external lane",
	})
}

func builtinThirdPartyTimeMCPServer() mcp.ServerDescriptor {
	return mcp.NormalizeServerDescriptor(mcp.ServerDescriptor{
		ID:               "third-party-time",
		Source:           "third-party",
		Enabled:          true,
		ToolCount:        1,
		ResourceCount:    0,
		Status:           "enabled",
		Command:          mcpThirdPartyTimeCommand(),
		Tools:            []string{"get_current_time"},
		Transport:        "stdio-third-party",
		ExecutionTier:    "canonical-verified",
		ExecutionSummary: "third-party time lane",
	})
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
		source := groupSource(group)
		localizationChecked := skillLocalizationChecked(root, id)
		isolationStatus := skillIsolationStatus(group, source)
		capabilityAudit := skill.VerifyCapability(root, id)
		items = append(items, skill.Descriptor{
			ID:                  id,
			Group:               group,
			Name:                humanizeName(id),
			Description:         groupDescription(group),
			Source:              source,
			VerificationStatus:  "implemented",
			LocalizationChecked: localizationChecked,
			IsolationStatus:     isolationStatus,
			CapabilityVerified:  capabilityAudit.Verified,
			CapabilitySummary:   capabilityAudit.Summary,
		})
	}
	return items
}

func skillLocalizationChecked(root string, id string) bool {
	return skillaudit.LocalizationChecked(root, id)
}

func skillIsolationStatus(group skill.Group, source string) string {
	switch group {
	case skill.Common:
		return skillIsolationStatusShared
	case skill.Codex:
		if source == "codex" {
			return skillIsolationStatusIsolated
		}
	case skill.CC:
		if source == "cc" {
			return skillIsolationStatusIsolated
		}
	}
	return skillIsolationStatusBlocked
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
		items = append(items, mcp.NormalizeServerDescriptor(mcp.ServerDescriptor{
			ID:            name,
			Source:        "node_modules",
			Enabled:       true,
			ToolCount:     0,
			ResourceCount: 0,
			Status:        "degraded",
		}))
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
		seen[item.ID] = mcp.NormalizeServerDescriptor(item)
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

func mcpFixtureCommand() []string {
	scriptPath := filepath.Join(workspaceRoot(), "scripts", "mcp_stdio_fixture.py")
	if goruntime.GOOS == "windows" {
		return []string{"python", scriptPath}
	}
	return []string{"python3", scriptPath}
}

func mcpSDKServerCommand() []string {
	scriptPath := filepath.Join(workspaceRoot(), "scripts", "mcp_sdk_server.js")
	return []string{"node", scriptPath}
}

func mcpThirdPartyTimeCommand() []string {
	scriptPath := filepath.Join(workspaceRoot(), "scripts", "mcp_third_party_time_server.js")
	return []string{"node", scriptPath}
}

func newServiceFromDiscoveryWithStore(discovered discoverySet, explicitStore *state.Store, providers *provider.Registry) *Service {
	return newServiceFromDiscovery(discovered, explicitStore, providers, true)
}

func newServiceFromDiscoveryWithStoreWithoutRecovery(discovered discoverySet, explicitStore *state.Store, providers *provider.Registry) *Service {
	return newServiceFromDiscovery(discovered, explicitStore, providers, false)
}

func newServiceFromDiscovery(discovered discoverySet, explicitStore *state.Store, providers *provider.Registry, recoverInterrupted bool) *Service {
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
			if recoverInterrupted {
				return NewService(defaultVersion, skill.Common, policy.DefaultMode(), projectRoot, registry, skills, mcpManager, providers, sessions)
			}
			return NewServiceWithoutRecovery(defaultVersion, skill.Common, policy.DefaultMode(), projectRoot, registry, skills, mcpManager, providers, sessions)
		}
		store = opened
	}
	sessions, err := session.NewRegistryWithStore(projectRoot, store)
	if err != nil {
		sessions = session.NewRegistry(projectRoot)
	}
	if recoverInterrupted {
		return NewService(defaultVersion, skill.Common, policy.DefaultMode(), projectRoot, registry, skills, mcpManager, providers, sessions)
	}
	return NewServiceWithoutRecovery(defaultVersion, skill.Common, policy.DefaultMode(), projectRoot, registry, skills, mcpManager, providers, sessions)
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

func SummarizeSkillGovernance(items []runtimecontract.Skill) []SkillGovernanceSummary {
	summaries := map[string]*SkillGovernanceSummary{
		"common": {Group: "common"},
		"codex":  {Group: "codex"},
		"cc":     {Group: "cc"},
	}

	for _, item := range items {
		group := strings.TrimSpace(item.Group)
		if group == "" {
			group = "common"
		}
		summary, ok := summaries[group]
		if !ok {
			summary = &SkillGovernanceSummary{Group: group}
			summaries[group] = summary
		}
		summary.ImplementedCount++
		if strings.EqualFold(strings.TrimSpace(item.VerificationStatus), "verified") {
			summary.VerifiedCount++
		}
		if !item.LocalizationChecked {
			summary.LocalizationPending++
		}
		if group != "common" && !item.CapabilityVerified {
			summary.CapabilityPending++
		}
	}

	order := []string{"common", "codex", "cc"}
	result := make([]SkillGovernanceSummary, 0, len(summaries))
	for _, group := range order {
		if summary, ok := summaries[group]; ok {
			result = append(result, *summary)
			delete(summaries, group)
		}
	}
	if len(summaries) > 0 {
		extras := make([]string, 0, len(summaries))
		for group := range summaries {
			extras = append(extras, group)
		}
		sort.Strings(extras)
		for _, group := range extras {
			result = append(result, *summaries[group])
		}
	}
	return result
}
