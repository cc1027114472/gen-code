package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadLocalToolsReturnsManifestTools(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.tools.json"), []byte(`{
  "tools": [
    {
      "name": "session-catchup",
      "description": "Recover planning context",
      "command": ["python", "scripts/session-catchup.py"],
      "readOnly": true
    }
  ]
}`), 0o644))

	items := LoadLocalTools(skillDir)
	require.Len(t, items, 1)
	require.Equal(t, "session-catchup", items[0].Name)
	require.Equal(t, []string{"python", "scripts/session-catchup.py"}, items[0].Command)
	require.True(t, items[0].ReadOnly)
}

func TestLoadLocalToolsIgnoresMissingManifest(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	items := LoadLocalTools(skillDir)
	require.Nil(t, items)
}
