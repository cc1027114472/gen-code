package skill

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGroup(t *testing.T) {
	group, err := ParseGroup("")
	require.NoError(t, err)
	require.Equal(t, Common, group)

	group, err = ParseGroup("codex")
	require.NoError(t, err)
	require.Equal(t, Codex, group)
}

func TestParseGroupInvalid(t *testing.T) {
	_, err := ParseGroup("other")
	require.EqualError(t, err, `invalid skill group "other"`)
}

func TestManagerListIncludesCommonAndTargetGroup(t *testing.T) {
	manager := NewManager([]Descriptor{
		{ID: "common.browser", Group: Common, Name: "Browser", Description: "Reusable browser skill"},
		{ID: "codex.review", Group: Codex, Name: "Review", Description: "Codex review flow"},
		{ID: "cc.swarm", Group: CC, Name: "Swarm", Description: "CC swarm flow"},
	})

	codexSkills := manager.List(Codex)
	require.Len(t, codexSkills, 2)
	require.ElementsMatch(t, []Group{Common, Codex}, []Group{codexSkills[0].Group, codexSkills[1].Group})

	ccSkills := manager.List(CC)
	require.Len(t, ccSkills, 2)
	require.ElementsMatch(t, []Group{Common, CC}, []Group{ccSkills[0].Group, ccSkills[1].Group})
}
