package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/mcp"
	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/skill"
	"llmtrace/internal/core/tool"
)

func TestDiscoverSkills(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "alpha-skill"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "beta-skill"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "gamma-skill.md"), []byte(""), 0o644))

	items := discoverSkills(root, skill.Codex)
	require.Len(t, items, 3)
	require.Equal(t, "alpha-skill", items[0].ID)
	require.Equal(t, skill.Codex, items[0].Group)
	require.Equal(t, "codex", items[0].Source)
	require.Equal(t, "implemented", items[0].VerificationStatus)
	require.Equal(t, "isolated", items[0].IsolationStatus)
	require.False(t, items[0].LocalizationChecked)
	require.Equal(t, "gamma-skill", items[2].ID)
}

func TestDiscoverSkillsMarksLocalizedMarkdownAsChecked(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "localized-skill.md"), []byte(`---
name: localized-skill
description: copied skill metadata can stay machine-readable
---

# 本地化技能

这是完整中文化审计样例。
- 先读取当前上下文。
- 再整理中文输出。
- 最后给出下一步建议。
`), 0o644))

	items := discoverSkills(root, skill.CC)
	require.Len(t, items, 1)
	require.True(t, items[0].LocalizationChecked)
	require.Equal(t, "isolated", items[0].IsolationStatus)
}

func TestDiscoverSkillsRejectsPartialChineseAsLocalized(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "mixed-skill.md"), []byte(`---
name: mixed-skill
description: metadata is ignored for localization audit
---

# 中文标题

This workflow is still explained in English.
这里只有一行中文补充。
`), 0o644))

	items := discoverSkills(root, skill.Codex)
	require.Len(t, items, 1)
	require.False(t, items[0].LocalizationChecked)
	require.Equal(t, "isolated", items[0].IsolationStatus)
}

func TestDiscoverSkillsIgnoresStructuralTagsAndQuadrupleCodeFences(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "tagged-skill.md"), []byte(
		"---\n"+
			"name: tagged-skill\n"+
			"description: metadata is ignored for localization audit\n"+
			"---\n\n"+
			"# 完整中文技能\n"+
			"<HARD-GATE>\n"+
			"这里是完整的中文说明。\n"+
			"</HARD-GATE>\n\n"+
			"````markdown\n"+
			"### Example\n"+
			"git commit -m \"feat: leave code examples untouched\"\n"+
			"````\n\n"+
			"- 继续保持中文步骤。\n"+
			"- 最终给出中文结论。\n",
	), 0o644))

	items := discoverSkills(root, skill.CC)
	require.Len(t, items, 1)
	require.True(t, items[0].LocalizationChecked)
	require.Equal(t, "isolated", items[0].IsolationStatus)
}

func TestDiscoverTools(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "tool-one"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "tool-two.js"), []byte(""), 0o644))

	items := discoverTools(root)
	require.Len(t, items, 2)
	require.Equal(t, "tool-one", items[0].ID)
	require.Equal(t, policy.AskUser, items[0].PermissionMode)
	require.Equal(t, "tool-two", items[1].ID)
}

func TestDiscoverMCPServers(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "@modelcontextprotocol"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "other"), 0o755))

	items := discoverMCPServers(root)
	require.Len(t, items, 1)
	require.Equal(t, "@modelcontextprotocol", items[0].ID)
	require.Equal(t, "degraded", items[0].Status)
	require.Equal(t, "@modelcontextprotocol (enabled, metadata health: degraded)", mcp.MetadataHealthSummary(items[0]))
}

func TestMCPMetadataHealthSummaryUsesStableLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		server      mcp.ServerDescriptor
		wantStatus  string
		wantSummary string
	}{
		{
			name: "enabled server stays enabled",
			server: mcp.ServerDescriptor{
				ID:            "filesystem",
				Enabled:       true,
				ToolCount:     2,
				ResourceCount: 1,
				Status:        "enabled",
			},
			wantStatus:  "enabled",
			wantSummary: "filesystem (enabled, metadata health: enabled)",
		},
		{
			name: "disabled server stays disabled",
			server: mcp.ServerDescriptor{
				ID:     "memory",
				Status: "enabled",
			},
			wantStatus:  "disabled",
			wantSummary: "memory (disabled, metadata health: disabled)",
		},
		{
			name: "zero inventory degrades enabled server",
			server: mcp.ServerDescriptor{
				ID:      "remote-proxy",
				Enabled: true,
			},
			wantStatus:  "degraded",
			wantSummary: "remote-proxy (enabled, metadata health: degraded)",
		},
		{
			name: "unreachable is preserved",
			server: mcp.ServerDescriptor{
				ID:            "stale-bridge",
				Enabled:       true,
				ToolCount:     1,
				ResourceCount: 0,
				Status:        "unreachable",
			},
			wantStatus:  "unreachable",
			wantSummary: "stale-bridge (enabled, metadata health: unreachable)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mcp.NormalizeServerDescriptor(tt.server)
			require.Equal(t, tt.wantStatus, got.Status)
			require.Equal(t, tt.wantSummary, mcp.MetadataHealthSummary(tt.server))
		})
	}
}

func TestDiscoverSiblingRuntimeContentUsesExpectedSiblingPaths(t *testing.T) {
	parent := t.TempDir()
	workspace := filepath.Join(parent, "gen-code")
	require.NoError(t, os.MkdirAll(filepath.Join(workspace, "internal", "core"), 0o755))

	require.NoError(t, os.MkdirAll(filepath.Join(parent, "codex", ".codex", "skills", "code-review"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "CC ibwhale", "ibwhale", ".claude", "skills", "writing-plans"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "CC ibwhale", "ibwhale", "tools", "deploy-helper"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "CC ibwhale", "node_modules", "@modelcontextprotocol", "server-filesystem"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "CC ibwhale", ".claude", "skills"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parent, "CC ibwhale", ".claude", "skills", "andrej-karpathy-skills.md"), []byte(""), 0o644))

	discovered := discoverSiblingRuntimeContent(workspace)

	require.Contains(t, discovered.skills, skill.Descriptor{
		ID:                  "common.browser",
		Group:               skill.Common,
		Name:                "Browser",
		Description:         "Common browser automation skill",
		Source:              "common",
		VerificationStatus:  "implemented",
		LocalizationChecked: true,
		IsolationStatus:     "shared-common",
	})
	require.Contains(t, discovered.skills, skill.Descriptor{
		ID:                  "code-review",
		Group:               skill.Codex,
		Name:                "Code Review",
		Description:         "Discovered from codex skills",
		Source:              "codex",
		VerificationStatus:  "implemented",
		LocalizationChecked: false,
		IsolationStatus:     "isolated",
	})
	require.Contains(t, discovered.skills, skill.Descriptor{
		ID:                  "andrej-karpathy-skills",
		Group:               skill.CC,
		Name:                "Andrej Karpathy Skills",
		Description:         "Discovered from cc skills",
		Source:              "cc",
		VerificationStatus:  "implemented",
		LocalizationChecked: false,
		IsolationStatus:     "isolated",
	})
	require.Contains(t, discovered.skills, skill.Descriptor{
		ID:                  "writing-plans",
		Group:               skill.CC,
		Name:                "Writing Plans",
		Description:         "Discovered from cc skills",
		Source:              "cc",
		VerificationStatus:  "implemented",
		LocalizationChecked: false,
		IsolationStatus:     "isolated",
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "deploy-helper",
		Name:               "Deploy Helper",
		Description:        "Discovered CC project tool",
		InputSchemaSummary: "Project-defined tool input",
		PermissionMode:     policy.AskUser,
		Source:             "cc",
		Kind:               "external",
		ReadOnly:           false,
		Executable:         false,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "mcp.tool.invoke",
		Name:               "MCP Tool Invoke",
		Description:        "Invoke a runtime-configured MCP tool through the shared task runner",
		InputSchemaSummary: `{"serverId":"external-fixture","toolName":"echo","arguments":{"message":"hello"}}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "mcp.tool.invoke",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.mcp, mcp.ServerDescriptor{
		ID:            "@modelcontextprotocol",
		Source:        "node_modules",
		Enabled:       true,
		ToolCount:     0,
		ResourceCount: 0,
		Status:        "degraded",
	})
	require.Contains(t, discovered.mcp, mcp.ServerDescriptor{
		ID:            "external-fixture",
		Source:        "fixture",
		Enabled:       true,
		ToolCount:     2,
		ResourceCount: 0,
		Status:        "enabled",
		Command:       mcpFixtureCommand(),
		Tools:         []string{"echo", "fail", "sum"},
	})
}

func TestNewSkillResolverPreservesGroupingSemantics(t *testing.T) {
	resolver := newSkillResolver(discoverySet{
		skills: []skill.Descriptor{
			{ID: "common.browser", Group: skill.Common},
			{ID: "codex.review", Group: skill.Codex},
			{ID: "cc.swarm", Group: skill.CC},
		},
	})

	require.Equal(t, []string{"common.browser"}, resolver.Common())

	codexGroup, ok := resolver.Resolve("codex")
	require.True(t, ok)
	require.Equal(t, []string{"codex.review", "common.browser"}, codexGroup.Skills)

	ccGroup, ok := resolver.Resolve("cc")
	require.True(t, ok)
	require.Equal(t, []string{"cc.swarm", "common.browser"}, ccGroup.Skills)
}

func TestSummarizeSkillGovernanceUsesStableGroupBaseline(t *testing.T) {
	summaries := SummarizeSkillGovernance([]runtimecontract.Skill{
		{ID: "common.browser", Group: "common", Source: "common", VerificationStatus: "implemented", LocalizationChecked: false},
		{ID: "codex.review", Group: "codex", Source: "codex", VerificationStatus: "verified", LocalizationChecked: true},
		{ID: "cc.swarm", Group: "cc", Source: "cc", VerificationStatus: "implemented", LocalizationChecked: false},
	})

	require.Len(t, summaries, 3)
	require.Equal(t, SkillGovernanceSummary{Group: "common", ImplementedCount: 1, VerifiedCount: 0, LocalizationPending: 1}, summaries[0])
	require.Equal(t, SkillGovernanceSummary{Group: "codex", ImplementedCount: 1, VerifiedCount: 1, LocalizationPending: 0}, summaries[1])
	require.Equal(t, SkillGovernanceSummary{Group: "cc", ImplementedCount: 1, VerifiedCount: 0, LocalizationPending: 1}, summaries[2])
}
