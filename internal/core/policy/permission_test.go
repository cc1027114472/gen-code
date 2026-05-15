package policy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePermissionModeDefaultsToAskUser(t *testing.T) {
	mode, err := ParsePermissionMode("")
	require.NoError(t, err)
	require.Equal(t, PermissionAskUser, mode)
}

func TestParsePermissionModeAcceptsKnownValues(t *testing.T) {
	cases := []struct {
		input string
		want  PermissionMode
	}{
		{input: "read-only", want: PermissionReadOnly},
		{input: "workspace-write", want: PermissionWorkspaceWrite},
		{input: "full-access", want: PermissionFullAccess},
		{input: "ask-user", want: PermissionAskUser},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			mode, err := ParsePermissionMode(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.want, mode)
		})
	}
}

func TestParsePermissionModeRejectsUnknownValues(t *testing.T) {
	_, err := ParsePermissionMode("unknown")
	require.EqualError(t, err, `invalid permission mode "unknown"`)
}
