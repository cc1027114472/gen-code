package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryListSortsAndPreservesDescriptorMetadata(t *testing.T) {
	registry := NewRegistry(Anthropic)
	registry.Register(Config{
		Kind:      Gemini,
		Enabled:   true,
		BaseURL:   "https://gemini.local",
		AuthToken: "gemini-token",
		Models: Models{
			Default: "gemini-2.5-pro",
		},
	})
	registry.Register(Config{
		Kind:      Anthropic,
		Enabled:   true,
		BaseURL:   "https://anthropic.local",
		AuthToken: "anthropic-token",
		Models: Models{
			Default: "claude-sonnet",
			Haiku:   "claude-haiku",
		},
	})

	items := registry.List()
	require.Len(t, items, 2)
	require.Equal(t, Anthropic, items[0].Kind)
	require.Equal(t, "claude-sonnet", items[0].DefaultModel)
	require.True(t, items[0].HasAuthToken)
	require.True(t, items[0].SupportsChat)
	require.False(t, items[0].SupportsTools)
	require.Equal(t, Gemini, items[1].Kind)
}

func TestRegistryDefaultFallsBackToFirstEnabledProvider(t *testing.T) {
	registry := NewRegistry("")
	registry.Register(Config{
		Kind:      OpenAI,
		Enabled:   true,
		BaseURL:   "https://openai.local",
		AuthToken: "openai-token",
		Models: Models{
			Default: "gpt-5",
		},
	})

	item, ok := registry.Default()
	require.True(t, ok)
	require.Equal(t, OpenAI, item.Kind)
	require.Equal(t, "gpt-5", item.DefaultModel)
}

func TestNormalizeKindCanonicalizesKnownProviders(t *testing.T) {
	require.Equal(t, Anthropic, NormalizeKind(" Anthropic "))
	require.Equal(t, OpenAI, NormalizeKind("OPENAI"))
	require.Equal(t, Gemini, NormalizeKind("gemini"))
	require.Equal(t, Kind("custom"), NormalizeKind("custom"))
}
