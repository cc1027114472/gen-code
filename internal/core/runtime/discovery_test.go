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

var governedProjectLocalSkillIDs = map[string][]string{
	"codex": {
		"babysit-pr",
		"architecture-blueprint-generator",
		"browser-use",
		"chrome",
		"code-review",
		"code-review-breaking-changes",
		"code-review-change-size",
		"code-review-context",
		"code-review-testing",
		"codex-bug",
		"codex-issue-digest",
		"codex-pr-body",
		"design-consultation",
		"frontend-design",
		"golang-backend-development",
		"imagegen",
		"kb-audit-governance-sync",
		"kb-audit-page-docs",
		"kb-audit-product-blueprint",
		"kb-audit-report-pack",
		"openai-docs",
		"plugin-creator",
		"remote-tests",
		"skill-creator",
		"skill-installer",
		"test-tui",
	},
	"cc": {
		"agent-browser",
		"andrej-karpathy-skills",
		"architecture-blueprint-generator",
		"breakdown-epic-arch",
		"breakdown-epic-pm",
		"breakdown-feature-prd",
		"canvas-design",
		"careful",
		"connect-chrome",
		"find-skills",
		"freeze",
		"guard",
		"create-implementation-plan",
		"brainstorming",
		"dispatching-parallel-agents",
		"executing-plans",
		"finishing-a-development-branch",
		"frontend-design",
		"go-backend-clean-architecture",
		"kb-audit-flow-prototype",
		"planning-with-files",
		"qa",
		"receiving-code-review",
		"ralph-loop",
		"react-vite-expert",
		"review",
		"requesting-code-review",
		"ship",
		"skill-creator",
		"subagent-driven-development",
		"systematic-debugging",
		"tailwindcss",
		"test-driven-development",
		"land-and-deploy",
		"setup-browser-cookies",
		"setup-deploy",
		"ui-ux-pro-max",
		"unfreeze",
		"using-git-worktrees",
		"use-my-browser",
		"using-superpowers",
		"vercel-react-best-practices",
		"verification-before-completion",
		"vite",
		"web-design-guidelines",
		"writing-plans",
		"writing-skills",
	},
}

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
	require.False(t, items[0].CapabilityVerified)
	require.Equal(t, "missing primary skill document", items[0].CapabilitySummary)
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
	require.True(t, items[0].CapabilityVerified)
	require.Equal(t, "capability verified", items[0].CapabilitySummary)
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
	require.True(t, items[0].CapabilityVerified)
	require.Equal(t, "capability verified", items[0].CapabilitySummary)
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
	require.True(t, items[0].CapabilityVerified)
	require.Equal(t, "capability verified", items[0].CapabilitySummary)
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

func TestDiscoverSiblingRuntimeContentUsesProjectLocalSkillCatalog(t *testing.T) {
	parent := t.TempDir()
	workspace := filepath.Join(parent, "gen-code")
	require.NoError(t, os.MkdirAll(filepath.Join(workspace, "internal", "core"), 0o755))

	require.NoError(t, os.MkdirAll(filepath.Join(workspace, "internal", "core", "skill", "catalog", "codex", "code-review"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspace, "internal", "core", "skill", "catalog", "cc", "writing-plans"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "internal", "core", "skill", "catalog", "cc", "andrej-karpathy-skills.md"), []byte(""), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "CC ibwhale", "ibwhale", "tools", "deploy-helper"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "CC ibwhale", "node_modules", "@modelcontextprotocol", "server-filesystem"), 0o755))

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
		CapabilityVerified:  false,
		CapabilitySummary:   "capability baseline not tracked for built-in shared common skill",
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
		CapabilityVerified:  false,
		CapabilitySummary:   "missing primary skill document",
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
		CapabilityVerified:  false,
		CapabilitySummary:   "missing frontmatter",
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
		CapabilityVerified:  false,
		CapabilitySummary:   "missing primary skill document",
	})
	for _, item := range discovered.skills {
		require.NotEqual(t, "gstack", item.ID)
	}
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
		InputSchemaSummary: `{"serverId":"sdk-external-fixture","toolName":"echo","arguments":{"message":"hello"}}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "mcp.tool.invoke",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "config.check_env",
		Name:               "Config Check Env",
		Description:        "Validate a workspace .env file against gen-code runtime parsing rules using tools/check_config.py",
		InputSchemaSummary: `{"envFile":"optional workspace-relative .env path","exampleFile":"optional workspace-relative .env.example path","strict":false}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "config.check_env",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "runtime.check_prerequisites",
		Name:               "Runtime Check Prerequisites",
		Description:        "Validate local workspace runtime prerequisites using tools/check_runtime.py",
		InputSchemaSummary: `{"workspace":"optional workspace root path","requireEnv":false,"strict":false}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "runtime.check_prerequisites",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "browser.state",
		Name:               "Browser State",
		Description:        "Inspect the current browser workspace state",
		InputSchemaSummary: "No input",
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.state",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "browser.open",
		Name:               "Browser Open",
		Description:        "Open a controlled browser tab for an allowlisted local URL, a managed authenticated session target, or a verified HTTPS read-only target",
		InputSchemaSummary: `{"url":"http://127.0.0.1:3000/"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.open",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "browser.navigate",
		Name:               "Browser Navigate",
		Description:        "Navigate an existing controlled browser tab to an allowlisted local URL, a managed authenticated session target, or a verified HTTPS read-only target",
		InputSchemaSummary: `{"tabId":"tab-1","url":"http://127.0.0.1:3000/"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.navigate",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "browser.click",
		Name:               "Browser Click",
		Description:        "Click a selector inside a controlled browser tab",
		InputSchemaSummary: `{"tabId":"tab-1","selector":"[data-testid='apply']"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.click",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "browser.type",
		Name:               "Browser Type",
		Description:        "Type text into a selector inside a controlled browser tab",
		InputSchemaSummary: `{"tabId":"tab-1","selector":"[data-testid='name']","text":"hello"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.type",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "browser.extract",
		Name:               "Browser Extract",
		Description:        "Extract text from a selector inside a controlled browser tab, authenticated fixture target, or verified HTTPS read-only target",
		InputSchemaSummary: `{"tabId":"tab-1","selector":"[data-testid='result']"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.extract",
		ReadOnly:           true,
		Executable:         true,
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "browser.screenshot",
		Name:               "Browser Screenshot",
		Description:        "Capture a screenshot for a controlled browser tab, authenticated fixture target, or verified HTTPS read-only target",
		InputSchemaSummary: `{"tabId":"tab-1"}`,
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
		Kind:               "browser.screenshot",
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
		Transport:     "stdio-fixture",
	})
	require.Contains(t, discovered.mcp, mcp.ServerDescriptor{
		ID:               "external-fixture",
		Source:           "fixture",
		Enabled:          true,
		ToolCount:        3,
		ResourceCount:    0,
		Status:           "enabled",
		Command:          mcpFixtureCommand(),
		Tools:            []string{"echo", "fail", "sum"},
		Transport:        "stdio-fixture",
		ExecutionTier:    "regression",
		ExecutionSummary: "fixture regression lane",
	})
	require.Contains(t, discovered.mcp, mcp.ServerDescriptor{
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
	require.Contains(t, discovered.mcp, mcp.ServerDescriptor{
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

func TestProjectLocalGovernedSkillCatalogIsFullyLocalized(t *testing.T) {
	workspace := workspaceRoot()
	discovered := discoverSiblingRuntimeContent(workspace)

	byGroupAndID := map[string]skill.Descriptor{}
	for _, item := range discovered.skills {
		byGroupAndID[string(item.Group)+":"+item.ID] = item
	}

	for group, ids := range governedProjectLocalSkillIDs {
		for _, id := range ids {
			key := group + ":" + id
			item, ok := byGroupAndID[key]
			require.Truef(t, ok, "expected discovered skill %s", key)
			require.Truef(t, item.LocalizationChecked, "expected project-local copied skill %s to pass localization audit", key)
			require.Equalf(t, group, item.Source, "expected source to remain stable for %s", key)
			require.Truef(t, item.CapabilityVerified, "expected project-local copied skill %s to pass capability audit", key)
			require.Equalf(t, "capability verified", item.CapabilitySummary, "expected project-local copied skill %s to report stable capability summary", key)
		}
	}
}

func TestGuardSiblingHookDependenciesResolveInProjectLocalCatalog(t *testing.T) {
	workspace := workspaceRoot()

	for _, path := range []string{
		filepath.Join(workspace, "internal", "core", "skill", "imports", "cc", "guard", "SKILL.md"),
		filepath.Join(workspace, "internal", "core", "skill", "catalog", "cc", "guard", "SKILL.md"),
		filepath.Join(workspace, "internal", "core", "skill", "catalog", "cc", "careful", "bin", "check-careful.sh"),
		filepath.Join(workspace, "internal", "core", "skill", "catalog", "cc", "freeze", "bin", "check-freeze.sh"),
		filepath.Join(workspace, "internal", "core", "skill", "imports", "cc", "careful", "bin", "check-careful.sh"),
		filepath.Join(workspace, "internal", "core", "skill", "imports", "cc", "freeze", "bin", "check-freeze.sh"),
	} {
		_, err := os.Stat(path)
		require.NoErrorf(t, err, "expected project-local guard dependency path to exist: %s", path)
	}
}

func TestHeavyGstackCopiedSkillDocumentsExistInImportsAndCatalog(t *testing.T) {
	workspace := workspaceRoot()

	for _, id := range []string{"setup-browser-cookies", "connect-chrome", "setup-deploy", "qa", "review", "ship", "land-and-deploy"} {
		for _, root := range []string{
			filepath.Join(workspace, "internal", "core", "skill", "imports", "cc"),
			filepath.Join(workspace, "internal", "core", "skill", "catalog", "cc"),
		} {
			_, err := os.Stat(filepath.Join(root, id, "SKILL.md"))
			require.NoErrorf(t, err, "expected copied skill document to exist for %s under %s", id, root)
		}
	}
}

func TestBrowseRemainsBlockedAsRuntimeHeavyGstackSurface(t *testing.T) {
	workspace := workspaceRoot()

	for _, path := range []string{
		filepath.Join(workspace, "internal", "core", "skill", "imports", "cc", "gstack", "browse", "SKILL.md"),
		filepath.Join(workspace, "internal", "core", "skill", "imports", "cc", "gstack", "browse", "SKILL.md.tmpl"),
		filepath.Join(workspace, "internal", "core", "skill", "imports", "cc", "gstack", "browse", "bin", "find-browse"),
		filepath.Join(workspace, "internal", "core", "skill", "imports", "cc", "gstack", "browse", "dist", "server-node.mjs"),
		filepath.Join(workspace, "internal", "core", "skill", "imports", "cc", "gstack", "browse", "dist", "bun-polyfill.cjs"),
	} {
		_, err := os.Stat(path)
		require.NoErrorf(t, err, "expected staged browse runtime-heavy evidence to exist: %s", path)
	}

	_, err := os.Stat(filepath.Join(workspace, "internal", "core", "skill", "catalog", "cc", "browse", "SKILL.md"))
	require.Error(t, err, "expected blocked browse skill to stay out of runtime-visible catalog")
	require.True(t, os.IsNotExist(err), "expected blocked browse catalog path to be absent, got: %v", err)

	for _, id := range []string{
		"autoplan",
		"benchmark",
		"canary",
		"codex",
		"cso",
		"design-consultation",
		"design-html",
		"design-review",
		"design-shotgun",
		"document-release",
		"gstack-upgrade",
		"investigate",
		"learn",
		"office-hours",
		"plan-ceo-review",
		"plan-design-review",
		"plan-eng-review",
		"qa-only",
		"retro",
	} {
		_, err := os.Stat(filepath.Join(workspace, "internal", "core", "skill", "catalog", "cc", id, "SKILL.md"))
		require.Error(t, err, "expected suite-only gstack surface %s to stay out of runtime-visible catalog", id)
		require.True(t, os.IsNotExist(err), "expected suite-only gstack surface %s to be absent from catalog", id)
	}
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
		{ID: "common.browser", Group: "common", Source: "common", VerificationStatus: "implemented", LocalizationChecked: false, CapabilityVerified: false},
		{ID: "codex.review", Group: "codex", Source: "codex", VerificationStatus: "verified", LocalizationChecked: true, CapabilityVerified: true},
		{ID: "cc.swarm", Group: "cc", Source: "cc", VerificationStatus: "implemented", LocalizationChecked: false, CapabilityVerified: false},
	})

	require.Len(t, summaries, 3)
	require.Equal(t, SkillGovernanceSummary{Group: "common", ImplementedCount: 1, VerifiedCount: 0, LocalizationPending: 1, CapabilityPending: 0}, summaries[0])
	require.Equal(t, SkillGovernanceSummary{Group: "codex", ImplementedCount: 1, VerifiedCount: 1, LocalizationPending: 0, CapabilityPending: 0}, summaries[1])
	require.Equal(t, SkillGovernanceSummary{Group: "cc", ImplementedCount: 1, VerifiedCount: 0, LocalizationPending: 1, CapabilityPending: 1}, summaries[2])
}
