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

func TestPolicyResolvesNamedSessionProfileForMultipleHosts(t *testing.T) {
	path := writePolicyFile(t, `{
  "profiles": {
    "acceptance-session": {
      "cookies": [
        {
          "name": "gc_auth",
          "value": "acceptance-session",
          "path": "/"
        }
      ]
    }
  },
  "hosts": {
    "127.0.0.1": {
      "sessionRequired": true,
      "profile": "acceptance-session"
    },
    "localhost": {
      "sessionRequired": true,
      "profile": "acceptance-session"
    }
  }
}`)

	policy := newPolicyFromSources("", path)

	firstProfile, needsSession, err := policy.sessionProfileForHost("127.0.0.1")
	require.NoError(t, err)
	require.True(t, needsSession)
	require.Len(t, firstProfile.Cookies, 1)
	require.Equal(t, "gc_auth", firstProfile.Cookies[0].Name)
	require.Equal(t, "acceptance-session", firstProfile.Cookies[0].Value)

	secondProfile, needsSession, err := policy.sessionProfileForHost("localhost")
	require.NoError(t, err)
	require.True(t, needsSession)
	require.Equal(t, firstProfile, secondProfile)
}

func TestPolicyFailsClosedWhenNamedSessionProfileMissing(t *testing.T) {
	path := writePolicyFile(t, `{
  "hosts": {
    "127.0.0.1": {
      "sessionRequired": true,
      "profile": "missing-profile"
    }
  }
}`)

	policy := newPolicyFromSources("", path)
	_, needsSession, err := policy.sessionProfileForHost("127.0.0.1")
	require.True(t, needsSession)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing-profile")
}

func TestPolicyFailsClosedWhenNamedSessionProfileCookiesAreInvalid(t *testing.T) {
	path := writePolicyFile(t, `{
  "profiles": {
    "broken-session": {
      "cookies": [
        {
          "name": "",
          "value": "broken"
        }
      ]
    }
  },
  "hosts": {
    "127.0.0.1": {
      "sessionRequired": true,
      "profile": "broken-session"
    }
  }
}`)

	policy := newPolicyFromSources("", path)
	_, needsSession, err := policy.sessionProfileForHost("127.0.0.1")
	require.True(t, needsSession)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid cookie")
}

func TestPolicyFileWithUTF8BOMLoadsSessionProfile(t *testing.T) {
	path := writePolicyFile(t, "\uFEFF{\n  \"profiles\": {\n    \"acceptance-session\": {\n      \"cookies\": [\n        {\n          \"name\": \"gc_auth\",\n          \"value\": \"acceptance-session\",\n          \"path\": \"/\"\n        }\n      ]\n    }\n  },\n  \"hosts\": {\n    \"127.0.0.1\": {\n      \"sessionRequired\": true,\n      \"profile\": \"acceptance-session\"\n    }\n  }\n}")

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
