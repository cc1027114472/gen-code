package policy

import (
	"fmt"
	"strings"
)

// PermissionMode is the public permission enum used by runtime policy checks.
type PermissionMode string

const (
	PermissionReadOnly       PermissionMode = "read-only"
	PermissionWorkspaceWrite PermissionMode = "workspace-write"
	PermissionFullAccess     PermissionMode = "full-access"
	PermissionAskUser        PermissionMode = "ask-user"
)

// DefaultPermissionMode returns the default runtime permission mode.
func DefaultPermissionMode() PermissionMode {
	return PermissionAskUser
}

// ParsePermissionMode converts user input into a known permission mode.
// Empty input resolves to the default ask-user mode.
func ParsePermissionMode(value string) (PermissionMode, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return DefaultPermissionMode(), nil
	}

	mode := PermissionMode(trimmed)
	switch mode {
	case PermissionReadOnly, PermissionWorkspaceWrite, PermissionFullAccess, PermissionAskUser:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid permission mode %q", value)
	}
}
