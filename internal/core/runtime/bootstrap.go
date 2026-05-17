package runtime

import (
	"sort"

	"llmtrace/internal/config"
	"llmtrace/internal/core/provider"
	"llmtrace/internal/core/state"
)

const defaultVersion = "0.1.0"

// NewDefaultService creates the minimal phase-one runtime used by server, CLI, and desktop.
func NewDefaultService() *Service {
	if cfg, err := config.Load(); err == nil {
		return NewDefaultServiceWithProviders(cfg.Providers)
	}
	discovered := discoverSiblingRuntimeContent(workspaceRoot())
	return newServiceFromDiscoveryWithStore(discovered, nil, nil)
}

// NewDefaultServiceWithoutRecovery creates the shared runtime without startup task recovery.
func NewDefaultServiceWithoutRecovery() *Service {
	if cfg, err := config.Load(); err == nil {
		return NewDefaultServiceWithProvidersWithoutRecovery(cfg.Providers)
	}
	discovered := discoverSiblingRuntimeContent(workspaceRoot())
	return newServiceFromDiscoveryWithStoreWithoutRecovery(discovered, nil, nil)
}

// NewDefaultServiceWithStateStore creates the shared runtime with an explicit state store.
func NewDefaultServiceWithStateStore(store *state.Store) *Service {
	discovered := discoverSiblingRuntimeContent(workspaceRoot())
	return newServiceFromDiscoveryWithStore(discovered, store, nil)
}

// NewDefaultServiceWithStateStoreWithoutRecovery creates the shared runtime without startup task recovery.
func NewDefaultServiceWithStateStoreWithoutRecovery(store *state.Store) *Service {
	discovered := discoverSiblingRuntimeContent(workspaceRoot())
	return newServiceFromDiscoveryWithStoreWithoutRecovery(discovered, store, nil)
}

// NewDefaultServiceWithProviders creates the shared runtime with provider configuration.
func NewDefaultServiceWithProviders(providers config.ProvidersConfig) *Service {
	discovered := discoverSiblingRuntimeContent(workspaceRoot())
	return newServiceFromDiscoveryWithStore(discovered, nil, newProviderRegistry(providers))
}

// NewDefaultServiceWithProvidersWithoutRecovery creates the shared runtime without startup task recovery.
func NewDefaultServiceWithProvidersWithoutRecovery(providers config.ProvidersConfig) *Service {
	discovered := discoverSiblingRuntimeContent(workspaceRoot())
	return newServiceFromDiscoveryWithStoreWithoutRecovery(discovered, nil, newProviderRegistry(providers))
}

func newProviderRegistry(cfg config.ProvidersConfig) *provider.Registry {
	registry := provider.NewRegistry(provider.Kind(cfg.DefaultProvider))
	registry.Register(provider.Config{
		Kind:      provider.Anthropic,
		Enabled:   cfg.Anthropic.Enabled,
		BaseURL:   cfg.Anthropic.BaseURL,
		AuthToken: cfg.Anthropic.AuthToken,
		Models: provider.Models{
			Default: cfg.Anthropic.Models.Default,
			Haiku:   cfg.Anthropic.Models.Haiku,
			Sonnet:  cfg.Anthropic.Models.Sonnet,
			Opus:    cfg.Anthropic.Models.Opus,
		},
	})
	registry.Register(provider.Config{
		Kind:      provider.OpenAI,
		Enabled:   cfg.OpenAI.Enabled,
		BaseURL:   cfg.OpenAI.BaseURL,
		AuthToken: cfg.OpenAI.AuthToken,
		Models: provider.Models{
			Default: cfg.OpenAI.Models.Default,
			Haiku:   cfg.OpenAI.Models.Haiku,
			Sonnet:  cfg.OpenAI.Models.Sonnet,
			Opus:    cfg.OpenAI.Models.Opus,
		},
	})
	registry.Register(provider.Config{
		Kind:      provider.Gemini,
		Enabled:   cfg.Gemini.Enabled,
		BaseURL:   cfg.Gemini.BaseURL,
		AuthToken: cfg.Gemini.AuthToken,
		Models: provider.Models{
			Default: cfg.Gemini.Models.Default,
			Haiku:   cfg.Gemini.Models.Haiku,
			Sonnet:  cfg.Gemini.Models.Sonnet,
			Opus:    cfg.Gemini.Models.Opus,
		},
	})
	return registry
}

// SkillGroups returns the concrete skill names grouped for CLI inspection.
func SkillGroups() map[string][]string {
	resolver := newSkillResolver(discoverSiblingRuntimeContent(workspaceRoot()))
	groups := map[string][]string{"common": resolver.Common()}
	if group, ok := resolver.Resolve("codex"); ok {
		groups["codex"] = append([]string(nil), group.Skills...)
	}
	if group, ok := resolver.Resolve("cc"); ok {
		groups["cc"] = append([]string(nil), group.Skills...)
	}

	for name, items := range groups {
		cloned := append([]string(nil), items...)
		sort.Strings(cloned)
		groups[name] = cloned
	}
	return groups
}
