package policy

import "fmt"

// Mode describes the allowed execution/approval policy for runtime operations.
type Mode string

const (
	ReadOnly       Mode = "read-only"
	WorkspaceWrite Mode = "workspace-write"
	FullAccess     Mode = "full-access"
	AskUser        Mode = "ask-user"
)

// DefaultMode returns the safe default used when no permission mode is specified.
func DefaultMode() Mode {
	return AskUser
}

// ParseMode resolves an input string into a supported Mode.
// Empty values fall back to the default mode.
func ParseMode(value string) (Mode, error) {
	if value == "" {
		return DefaultMode(), nil
	}

	mode := Mode(value)
	switch mode {
	case ReadOnly, WorkspaceWrite, FullAccess, AskUser:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid permission mode %q", value)
	}
}
