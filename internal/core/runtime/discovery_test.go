package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

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
	require.Equal(t, "gamma-skill", items[2].ID)
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
		ID:          "common.browser",
		Group:       skill.Common,
		Name:        "Browser",
		Description: "Common browser automation skill",
	})
	require.Contains(t, discovered.skills, skill.Descriptor{
		ID:          "code-review",
		Group:       skill.Codex,
		Name:        "Code Review",
		Description: "Discovered from codex skills",
	})
	require.Contains(t, discovered.skills, skill.Descriptor{
		ID:          "andrej-karpathy-skills",
		Group:       skill.CC,
		Name:        "Andrej Karpathy Skills",
		Description: "Discovered from cc skills",
	})
	require.Contains(t, discovered.skills, skill.Descriptor{
		ID:          "writing-plans",
		Group:       skill.CC,
		Name:        "Writing Plans",
		Description: "Discovered from cc skills",
	})
	require.Contains(t, discovered.tools, tool.Descriptor{
		ID:                 "deploy-helper",
		Name:               "Deploy Helper",
		Description:        "Discovered CC project tool",
		InputSchemaSummary: "Project-defined tool input",
		PermissionMode:     policy.AskUser,
		Source:             "cc",
	})
	require.Contains(t, discovered.mcp, mcp.ServerDescriptor{
		ID:            "@modelcontextprotocol",
		Source:        "node_modules",
		Enabled:       true,
		ToolCount:     0,
		ResourceCount: 0,
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
