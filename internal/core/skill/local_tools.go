package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type localToolManifest struct {
	Tools []LocalToolDescriptor `json:"tools"`
}

func LoadLocalTools(skillDir string) []LocalToolDescriptor {
	if strings.TrimSpace(skillDir) == "" {
		return nil
	}
	manifestPath := filepath.Join(skillDir, "skill.tools.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}
	var manifest localToolManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return nil
	}
	tools := make([]LocalToolDescriptor, 0, len(manifest.Tools))
	for _, item := range manifest.Tools {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		tools = append(tools, item)
	}
	return tools
}
