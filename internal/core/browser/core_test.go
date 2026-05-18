package browser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeURLRejectsDisallowedURL(t *testing.T) {
	_, err := normalizeURL("https://example.com")
	require.ErrorIs(t, err, ErrURLNotAllowed)
}

func TestNormalizeURLAcceptsControlledLocalURL(t *testing.T) {
	value, err := normalizeURL("127.0.0.1:40123")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:40123", value)
}

func TestDriverStateStartsEmpty(t *testing.T) {
	driver := NewDriver()
	snapshot, err := driver.State(context.Background())
	require.NoError(t, err)
	require.Empty(t, snapshot.ActiveTabID)
	require.Empty(t, snapshot.Tabs)
}

func TestDriverOpenRejectsDisallowedURL(t *testing.T) {
	driver := NewDriver()
	snapshot, err := driver.Open(context.Background(), OpenRequest{URL: "https://example.com"})
	require.ErrorIs(t, err, ErrURLNotAllowed)
	require.Empty(t, snapshot.ActiveTabID)
	require.Empty(t, snapshot.Tabs)
	require.Contains(t, snapshot.LatestActionError, ErrURLNotAllowed.Error())
}
