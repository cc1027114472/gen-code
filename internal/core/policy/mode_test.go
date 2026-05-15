package policy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultMode(t *testing.T) {
	require.Equal(t, AskUser, DefaultMode())
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Mode
	}{
		{name: "empty defaults to ask", input: "", want: AskUser},
		{name: "read only", input: "read-only", want: ReadOnly},
		{name: "workspace write", input: "workspace-write", want: WorkspaceWrite},
		{name: "full access", input: "full-access", want: FullAccess},
		{name: "ask user", input: "ask-user", want: AskUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMode(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseModeInvalid(t *testing.T) {
	_, err := ParseMode("danger-mode")
	require.EqualError(t, err, `invalid permission mode "danger-mode"`)
}
