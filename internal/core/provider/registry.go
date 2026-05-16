package provider

import (
	"fmt"
	"sort"
	"sync"
)

// Registry stores provider configs and resolves the current default provider.
type Registry struct {
	mu              sync.RWMutex
	defaultProvider Kind
	providers       map[Kind]Config
}

// NewRegistry constructs an empty provider registry.
func NewRegistry(defaultProvider Kind) *Registry {
	return &Registry{
		defaultProvider: NormalizeKind(string(defaultProvider)),
		providers:       make(map[Kind]Config),
	}
}

// Register stores or replaces a provider config.
func (r *Registry) Register(cfg Config) {
	r.mu.Lock()
	defer r.mu.Unlock()

	kind := NormalizeKind(string(cfg.Kind))
	cfg.Kind = kind
	r.providers[kind] = cfg
	if r.defaultProvider == "" && cfg.Enabled {
		r.defaultProvider = kind
	}
}

// Default returns the current default provider descriptor.
func (r *Registry) Default() (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, ok := r.providers[r.defaultProvider]
	if !ok {
		return Descriptor{}, false
	}
	return cfg.Descriptor(), true
}

// Resolve returns a provider config by kind.
func (r *Registry) Resolve(kind Kind) (Config, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, ok := r.providers[NormalizeKind(string(kind))]
	return cfg, ok
}

// ResolveDefault returns the preferred enabled provider config.
func (r *Registry) ResolveDefault() (Config, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cfg, ok := r.providers[r.defaultProvider]; ok && cfg.Enabled {
		return cfg, true
	}

	kinds := make([]Kind, 0, len(r.providers))
	for kind := range r.providers {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool {
		return kinds[i] < kinds[j]
	})
	for _, kind := range kinds {
		cfg := r.providers[kind]
		if cfg.Enabled {
			return cfg, true
		}
	}
	return Config{}, false
}

// ResolveForExecution returns an enabled provider config by explicit kind or default fallback.
func (r *Registry) ResolveForExecution(kind string) (Config, error) {
	if normalized := NormalizeKind(kind); normalized != "" {
		cfg, ok := r.Resolve(normalized)
		if !ok {
			return Config{}, fmt.Errorf("provider not found: %s", kind)
		}
		if !cfg.Enabled {
			return Config{}, fmt.Errorf("provider disabled: %s", kind)
		}
		return cfg, nil
	}

	cfg, ok := r.ResolveDefault()
	if !ok {
		return Config{}, fmt.Errorf("no enabled provider configured")
	}
	return cfg, nil
}

// List returns all provider descriptors sorted by kind.
func (r *Registry) List() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Descriptor, 0, len(r.providers))
	for _, cfg := range r.providers {
		items = append(items, cfg.Descriptor())
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Kind < items[j].Kind
	})
	return items
}
