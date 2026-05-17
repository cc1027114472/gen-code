package skill

import (
	"fmt"
	"sort"
)

// Group identifies a logical skill bundle.
type Group string

const (
	Common Group = "common"
	Codex  Group = "codex"
	CC     Group = "cc"
)

// Descriptor describes a discoverable skill.
type Descriptor struct {
	ID                  string `json:"id"`
	Group               Group  `json:"group"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Source              string `json:"source"`
	VerificationStatus  string `json:"verificationStatus"`
	LocalizationChecked bool   `json:"localizationChecked"`
	IsolationStatus     string `json:"isolationStatus"`
}

// ParseGroup resolves a textual group identifier. Empty values default to Common.
func ParseGroup(value string) (Group, error) {
	if value == "" {
		return Common, nil
	}

	group := Group(value)
	switch group {
	case Common, Codex, CC:
		return group, nil
	default:
		return "", fmt.Errorf("invalid skill group %q", value)
	}
}

// Manager stores skills grouped by runtime ownership.
type Manager struct {
	skills []Descriptor
}

// NewManager constructs a new Manager.
func NewManager(skills []Descriptor) *Manager {
	cloned := make([]Descriptor, len(skills))
	copy(cloned, skills)
	return &Manager{skills: cloned}
}

// List returns skills visible for the requested group.
// Common skills are always included.
func (m *Manager) List(group Group) []Descriptor {
	items := make([]Descriptor, 0, len(m.skills))
	for _, item := range m.skills {
		if item.Group == Common || item.Group == group {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Group == items[j].Group {
			return items[i].ID < items[j].ID
		}
		return items[i].Group < items[j].Group
	})
	return items
}
