package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyCapabilityPassesDirectorySkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "references"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "references", "guide.md"), []byte("# guide\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: demo-skill
description: demo description
---

## Goal

Use this skill to review the target changes.

- Read the current diff.
- Check the linked reference.

See [guide](references/guide.md).
`), 0o644))

	audit := VerifyCapability(root, "demo-skill")
	require.True(t, audit.Verified)
	require.Equal(t, "capability verified", audit.Summary)
}

func TestVerifyCapabilityPassesSingleFileSkill(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "single.md"), []byte(`---
name: single
description: single file skill
---

这是一个可复用的单文件 skill。
请先检查输入，再返回结果。`), 0o644))

	audit := VerifyCapability(root, "single")
	require.True(t, audit.Verified)
	require.Equal(t, "capability verified", audit.Summary)
}

func TestVerifyCapabilityRejectsMissingReference(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: demo-skill
description: demo description
---

## Steps

- Read the reference first.

See [guide](references/guide.md).
`), 0o644))

	audit := VerifyCapability(root, "demo-skill")
	require.False(t, audit.Verified)
	require.Equal(t, "missing referenced file: references/guide.md", audit.Summary)
}

func TestVerifyCapabilityRejectsBadFrontmatter(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "broken.md"), []byte(`# no frontmatter`), 0o644))

	audit := VerifyCapability(root, "broken")
	require.False(t, audit.Verified)
	require.Equal(t, "missing frontmatter", audit.Summary)
}

func TestVerifyCapabilityRejectsMissingCapabilityStructure(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "flat.md"), []byte(`---
name: flat
description: flat skill
---

简短说明。`), 0o644))

	audit := VerifyCapability(root, "flat")
	require.False(t, audit.Verified)
	require.Equal(t, "missing capability structure", audit.Summary)
}

func TestVerifyCapabilityAcceptsNarrativeCapabilityStructureWithoutMarkdownLists(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "narrative.md"), []byte(`---
name: narrative
description: narrative skill
---

使用这个 skill 时，先检查当前输入是否完整，再读取相关上下文，并返回带编号的结果说明。
如果发现约束冲突，必须明确指出原因，并说明下一步应该如何继续。`), 0o644))

	audit := VerifyCapability(root, "narrative")
	require.True(t, audit.Verified)
	require.Equal(t, "capability verified", audit.Summary)
}
