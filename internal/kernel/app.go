package kernel

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// App is the plug board every module registers into. It knows nothing about
// AI completions, billing, or auth specifically — it only knows how to hold
// named things and hand them back out. All real behavior lives in modules.
type App struct {
	mu sync.RWMutex

	Config        *Config
	Events        *EventBus
	Store         *Store
	MeteringChain *MeteringChain // multi-backend metering storage orchestrator
	Mux           *http.ServeMux // shared HTTP router every HTTP-facing module registers routes into

	encoders map[string]Encoder
	messages map[string]MessageDescriptor
	handlers map[string]MessageHandler
	policies map[string]Policy
	modules  []Module
}

func NewApp(cfg *Config) *App {
	app := &App{
		Config:   cfg,
		Events:   NewEventBus(),
		Store:    NewStore(),
		Mux:      http.NewServeMux(),
		encoders: make(map[string]Encoder),
		messages: make(map[string]MessageDescriptor),
		handlers: make(map[string]MessageHandler),
		policies: make(map[string]Policy),
	}

	// Build the MeteringChain from configuration.
	chain, err := buildMeteringChain(cfg)
	if err != nil {
		// Log but don't crash — the chain will be nil and Store delegates
		// will be no-ops. This allows tests with minimal config to work.
		fmt.Printf("[app] WARNING: failed to build metering chain: %v\n", err)
	} else {
		app.MeteringChain = chain
		app.Store.SetMeteringChain(chain)
		chain.Start()
	}

	return app
}

// buildMeteringChain constructs the chain from Config, adding backends
// based on the METERING_BACKENDS setting.
func buildMeteringChain(cfg *Config) (*MeteringChain, error) {
	// Resolve WAL directory.
	walDir := cfg.MeteringWALDir
	if walDir == "" {
		walDir = filepath.Join(cfg.DataDir, "wal")
	}

	// Apply defaults for zero-value fields (e.g., when tests pass a minimal Config{}).
	if cfg.MeteringChannelSize <= 0 {
		cfg.MeteringChannelSize = 1000
	}
	if cfg.MeteringDedupRetentionMs <= 0 {
		cfg.MeteringDedupRetentionMs = 3600000
	}
	if cfg.MeteringCBThreshold <= 0 {
		cfg.MeteringCBThreshold = 5
	}
	if cfg.MeteringCBCooldownMs <= 0 {
		cfg.MeteringCBCooldownMs = 30000
	}
	if cfg.MeteringWALRetryMs <= 0 {
		cfg.MeteringWALRetryMs = 5000
	}

	chainCfg := MeteringChainConfig{
		WAL: WALConfig{
			Dir:            walDir,
			MaxSegmentSize: 10 * 1024 * 1024, // 10 MB
			MaxSegmentAge:  10 * time.Minute,
			RetryInterval:  time.Duration(cfg.MeteringWALRetryMs) * time.Millisecond,
			Enabled:        cfg.MeteringWALEnabled,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Threshold: cfg.MeteringCBThreshold,
			Cooldown:  time.Duration(cfg.MeteringCBCooldownMs) * time.Millisecond,
		},
		ChannelSize:    cfg.MeteringChannelSize,
		DedupRetention: time.Duration(cfg.MeteringDedupRetentionMs) * time.Millisecond,
	}

	chain, err := NewMeteringChain(chainCfg)
	if err != nil {
		return nil, fmt.Errorf("create chain: %w", err)
	}

	// Parse configured backends and add them. Always ensure memory (L1) is present.
	backendsStr := strings.TrimSpace(cfg.MeteringBackends)
	if backendsStr == "" {
		backendsStr = "memory"
	}

	memAdded := false
	for _, name := range strings.Split(backendsStr, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		switch name {
		case "memory":
			chain.AddBackend(NewMemoryMeteringStore())
			memAdded = true
		case "postgres":
			if cfg.MeteringPostgresDSN != "" {
				chain.AddBackend(NewPostgresMeteringStore(cfg.MeteringPostgresDSN))
			}
		case "redis":
			if cfg.MeteringRedisAddr != "" {
				chain.AddBackend(NewRedisMeteringStore(cfg.MeteringRedisAddr))
			}
		default:
			// Unknown backend name — skip with warning (only if non-empty).
			fmt.Printf("[app] WARNING: unknown metering backend %q, skipping\n", name)
		}
	}

	// Guarantee at least the memory backend is always present.
	if !memAdded {
		chain.AddBackend(NewMemoryMeteringStore())
	}

	return chain, nil
}

