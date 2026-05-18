package browser

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureSessionBootstrapAppliesCookiesForMatchingHost(t *testing.T) {
	policy := newPolicyFromSources("", writePolicyFile(t, `{
  "hosts": {
    "localhost": {
      "sessionRequired": true,
      "cookies": [
        {
          "name": "session_id",
          "value": "abc123",
          "path": "/"
        }
      ]
    }
  }
}`))
	driver := newDriverWithPolicy(policy)
	tab := &tabSession{id: "browser-tab-1"}
	var calls int
	driver.applySessionCookies = func(_ context.Context, _ *tabSession, target *url.URL, profile SessionProfile) error {
		calls++
		require.Equal(t, "localhost", target.Hostname())
		require.Len(t, profile.Cookies, 1)
		require.Equal(t, "session_id", profile.Cookies[0].Name)
		return nil
	}

	err := driver.ensureSessionBootstrap(context.Background(), tab, "http://localhost:4173/app")
	require.NoError(t, err)
	require.Equal(t, 1, calls)

	err = driver.ensureSessionBootstrap(context.Background(), tab, "http://localhost:4173/settings")
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

func TestEnsureSessionBootstrapSkipsHostsWithoutSessionProfile(t *testing.T) {
	policy := newPolicyFromSources("example.com", "")
	driver := newDriverWithPolicy(policy)
	tab := &tabSession{id: "browser-tab-1"}
	driver.applySessionCookies = func(_ context.Context, _ *tabSession, _ *url.URL, _ SessionProfile) error {
		t.Fatalf("unexpected cookie bootstrap")
		return nil
	}

	err := driver.ensureSessionBootstrap(context.Background(), tab, "https://example.com/dashboard")
	require.NoError(t, err)
}

func TestEnsureSessionBootstrapFailsOnInvalidCookieConfig(t *testing.T) {
	policy := newPolicyFromSources("", writePolicyFile(t, `{
  "hosts": {
    "localhost": {
      "sessionRequired": true,
      "cookies": [
        {
          "name": "",
          "value": "broken"
        }
      ]
    }
  }
}`))
	driver := newDriverWithPolicy(policy)

	err := driver.ensureSessionBootstrap(context.Background(), &tabSession{id: "browser-tab-1"}, "http://localhost:4173/app")
	require.ErrorIs(t, err, ErrSessionUnavailable)
}

func TestEnsureSessionBootstrapFailsClosedWhenProfileMissing(t *testing.T) {
	policy := newPolicyFromSources("", writePolicyFile(t, `{
  "hosts": {
    "localhost": {
      "sessionRequired": true
    }
  }
}`))
	driver := newDriverWithPolicy(policy)

	err := driver.ensureSessionBootstrap(context.Background(), &tabSession{id: "browser-tab-1"}, "http://localhost:4173/app")
	require.ErrorIs(t, err, ErrSessionUnavailable)
}
