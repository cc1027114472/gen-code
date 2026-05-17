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

正文。
`), 0o644))

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
