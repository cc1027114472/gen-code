package tool

import (
	"testing"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/core/policy"
)

func TestRegistryRegisterAndList(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Descriptor{
		ID:                 "bridge.check",
		Name:               "Bridge Check",
		Description:        "Verify the local bridge is available",
		InputSchemaSummary: "No input",
		PermissionMode:     policy.AskUser,
		Source:             "runtime",
	})
	registry.Register(Descriptor{
		ID:                 "skills.list",
		Name:               "Skills List",
		Description:        "List available skill groups",
		InputSchemaSummary: "Optional group filter",
		PermissionMode:     policy.ReadOnly,
		Source:             "runtime",
	})

	items := registry.List()
	require.Len(t, items, 2)
	require.Equal(t, "bridge.check", items[0].ID)
	require.Equal(t, policy.AskUser, items[0].PermissionMode)
	require.Equal(t, "runtime", items[0].Source)
	require.Equal(t, "skills.list", items[1].ID)
	require.Equal(t, policy.ReadOnly, items[1].PermissionMode)
	require.Equal(t, "runtime", items[1].Source)
}
