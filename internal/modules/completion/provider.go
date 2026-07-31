package completion

import (
	"context"
	"fmt"
	"sync"
)

// CompletionProvider is the strategy interface for AI completion backends.
type CompletionProvider interface {
	Name() string
	Generate(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)
}

// ProviderRegistry manages registered completion providers and fallback logic.
type ProviderRegistry struct {
	mu          sync.RWMutex
	providers   map[string]CompletionProvider
	defaultName string
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers:   make(map[string]CompletionProvider),
		defaultName: "mock",
	}
}

// Register registers a provider implementation under its Name().
func (r *ProviderRegistry) Register(p CompletionProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// SetDefault sets the fallback default provider name.
func (r *ProviderRegistry) SetDefault(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultName = name
}

// Get fetches a provider by name.
func (r *ProviderRegistry) Get(name string) (CompletionProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// GetDefault returns the default provider.
func (r *ProviderRegistry) GetDefault() CompletionProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.providers[r.defaultName]; ok {
		return p
	}
	// Fallback to any registered provider if default isn't found
	for _, p := range r.providers {
		return p
	}
	return nil
}

// GenerateWithFallback executes completion using preferred provider, falling back to default/mock if primary fails.
func (r *ProviderRegistry) GenerateWithFallback(ctx context.Context, preferredName string, req *ChatRequest) (*ChatResponse, string, error) {
	var primary CompletionProvider
	if preferredName != "" {
		if p, ok := r.Get(preferredName); ok {
			primary = p
		}
	}
	if primary == nil {
		primary = r.GetDefault()
	}

	if primary == nil {
		return nil, "", ErrProviderUnavailable
	}

	resp, err := primary.Generate(ctx, req)
	if err == nil {
		return resp, primary.Name(), nil
	}

	// Try fallback to mock if primary failed and wasn't mock already
	if primary.Name() != "mock" {
		if fallback, ok := r.Get("mock"); ok {
			resp, errFallback := fallback.Generate(ctx, req)
			if errFallback == nil {
				return resp, fallback.Name(), nil
			}
		}
	}

	return nil, primary.Name(), fmt.Errorf("provider %s failed: %w", primary.Name(), err)
}

// StreamWithFallback executes streaming completion with automatic provider fallback.
func (r *ProviderRegistry) StreamWithFallback(ctx context.Context, preferredName string, req *ChatRequest) (<-chan StreamChunk, string, error) {
	var primary CompletionProvider
	if preferredName != "" {
		if p, ok := r.Get(preferredName); ok {
			primary = p
		}
	}
	if primary == nil {
		primary = r.GetDefault()
	}

	if primary == nil {
		return nil, "", ErrProviderUnavailable
	}

	ch, err := primary.Stream(ctx, req)
	if err == nil {
		return ch, primary.Name(), nil
	}

	// Try fallback to mock if primary stream failed
	if primary.Name() != "mock" {
		if fallback, ok := r.Get("mock"); ok {
			chFallback, errFallback := fallback.Stream(ctx, req)
			if errFallback == nil {
				return chFallback, fallback.Name(), nil
			}
		}
	}

	return nil, primary.Name(), fmt.Errorf("stream provider %s failed: %w", primary.Name(), err)
}
