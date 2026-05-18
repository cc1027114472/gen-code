package browser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyAllowsAllowlistedHTTPSHost(t *testing.T) {
	policy := newPolicyFromSources("example.com", "")

	value, err := normalizeURLWithPolicy("https://example.com/dashboard", policy)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/dashboard", value)
}

func TestPolicyRejectsMalformedAllowlistEntries(t *testing.T) {
	policy := newPolicyFromSources("https://example.com/path", "")

	_, err := normalizeURLWithPolicy("https://example.com/dashboard", policy)
	require.ErrorIs(t, err, ErrURLNotAllowed)
}

func TestPolicyAllowsLocalhostWithoutAllowlist(t *testing.T) {
	policy := newPolicyFromSources("", "")

	value, err := normalizeURLWithPolicy("localhost:4173", policy)
	require.NoError(t, err)
	require.Equal(t, "http://localhost:4173", value)
}

func TestPolicyFileWithUTF8BOMLoadsSessionProfile(t *testing.T) {
	path := writePolicyFile(t, "\uFEFF{\n  \"hosts\": {\n    \"127.0.0.1\": {\n      \"sessionRequired\": true,\n      \"cookies\": [\n        {\n          \"name\": \"gc_auth\",\n          \"value\": \"acceptance-session\",\n          \"path\": \"/\"\n        }\n      ]\n    }\n  }\n}")

	policy := newPolicyFromSources("", path)
	profile, needsSession, err := policy.sessionProfileForHost("127.0.0.1")
	require.NoError(t, err)
	require.True(t, needsSession)
	require.Len(t, profile.Cookies, 1)
	require.Equal(t, "gc_auth", profile.Cookies[0].Name)
	require.Equal(t, "acceptance-session", profile.Cookies[0].Value)
}

func writePolicyFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "browser-policy.json")
	err := os.WriteFile(path, []byte(body), 0o600)
	require.NoError(t, err)
	return path
}
