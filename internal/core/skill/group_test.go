package skill

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolverReusesCommonSkills(t *testing.T) {
	resolver := NewResolver([]string{"common-a", "common-b"}, map[string][]string{
		"builder": []string{"builder-a", "common-b"},
	})

	group, ok := resolver.Resolve("builder")
	require.True(t, ok)
	require.Equal(t, "builder", group.Name)
	require.True(t, group.UsesCommon)
	require.Equal(t, []string{"builder-a", "common-a", "common-b"}, group.Skills)
}

func TestResolverReturnsMissingGroup(t *testing.T) {
	resolver := NewResolver([]string{"common-a"}, map[string][]string{})

	_, ok := resolver.Resolve("missing")
	require.False(t, ok)
}
