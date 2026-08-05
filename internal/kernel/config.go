package kernel

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config is intentionally a flat struct — every module reads only the
// fields it needs off one shared, already-resolved source of truth.
type Config struct {
	HTTPPort        string
	LogLevel        string
	DailyTokenQuota int
	DataDir         string
	LocalTest       bool

	// --- Metering storage configuration ---
	MeteringBackends         string // comma-separated: "memory,postgres,redis" (default: "memory")
	MeteringPostgresDSN      string // PostgreSQL connection string (enables postgres backend)
	MeteringRedisAddr        string // Redis address (enables redis backend)
	MeteringWALEnabled       bool   // enable WAL for write failure recovery (default: true)
	MeteringWALDir           string // WAL directory (default: "{DataDir}/wal")
	MeteringWALRetryMs       int    // WAL retry interval in milliseconds (default: 5000)
	MeteringCBThreshold      int    // circuit breaker failure threshold (default: 5)
	MeteringCBCooldownMs     int    // circuit breaker cooldown in milliseconds (default: 30000)
	MeteringChannelSize      int    // async write channel capacity (default: 10000)
	MeteringDedupRetentionMs int    // dedup window in milliseconds (default: 3600000 = 1h)
	MeteringBatchSize        int    // batch flush size for slow backends (default: 100)
	MeteringBatchFlushMs     int    // batch flush interval in milliseconds (default: 1000)
}

// LoadConfig layers three sources, lowest to highest priority:
// 1. hardcoded defaults, 2. config.env file (simple KEY=VALUE lines,
// dependency-free so this lab needs no YAML library), 3. real env vars.
func LoadConfig(path string) *Config {
	cfg := &Config{
		HTTPPort:        "8080",
		LogLevel:        "info",
		DailyTokenQuota: 5000,
		DataDir:         "./data",
		LocalTest:       false,

		// Metering defaults — memory-only, WAL enabled, sensible thresholds.
		MeteringBackends:         "memory",
		MeteringWALEnabled:       true,
		MeteringWALRetryMs:       5000,
		MeteringCBThreshold:      5,
		MeteringCBCooldownMs:     30000,
		MeteringChannelSize:      10000,
		MeteringDedupRetentionMs: 3600000, // 1 hour
		MeteringBatchSize:        100,
		MeteringBatchFlushMs:     1000,
	}

	if f, err := os.Open(path); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			applyConfigValue(cfg, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	envKeys := []string{
		"HTTP_PORT", "LOG_LEVEL", "DAILY_TOKEN_QUOTA", "DATA_DIR", "LOCAL_TEST",
		"METERING_BACKENDS", "METERING_POSTGRES_DSN", "METERING_REDIS_ADDR",
		"METERING_WAL_ENABLED", "METERING_WAL_DIR", "METERING_WAL_RETRY_MS",
		"METERING_CB_THRESHOLD", "METERING_CB_COOLDOWN_MS",
		"METERING_CHANNEL_SIZE", "METERING_DEDUP_RETENTION_MS",
		"METERING_BATCH_SIZE", "METERING_BATCH_FLUSH_MS",
	}
	for _, key := range envKeys {
		if v := os.Getenv(key); v != "" {
			applyConfigValue(cfg, key, v)
		}
	}

	return cfg
}

func applyConfigValue(cfg *Config, key, val string) {
	switch key {
	case "HTTP_PORT":
		cfg.HTTPPort = val
	case "LOG_LEVEL":
		cfg.LogLevel = val
	case "DAILY_TOKEN_QUOTA":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.DailyTokenQuota = n
		}
	case "DATA_DIR":
		cfg.DataDir = val
	case "LOCAL_TEST":
		cfg.LocalTest = val == "1" || strings.EqualFold(val, "true") || strings.EqualFold(val, "yes")

	// --- Metering storage ---
	case "METERING_BACKENDS":
		cfg.MeteringBackends = val
	case "METERING_POSTGRES_DSN":
		cfg.MeteringPostgresDSN = val
	case "METERING_REDIS_ADDR":
		cfg.MeteringRedisAddr = val
	case "METERING_WAL_ENABLED":
		cfg.MeteringWALEnabled = val == "1" || strings.EqualFold(val, "true") || strings.EqualFold(val, "yes")
	case "METERING_WAL_DIR":
		cfg.MeteringWALDir = val
	case "METERING_WAL_RETRY_MS":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.MeteringWALRetryMs = n
		}
	case "METERING_CB_THRESHOLD":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.MeteringCBThreshold = n
		}
	case "METERING_CB_COOLDOWN_MS":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.MeteringCBCooldownMs = n
		}
	case "METERING_CHANNEL_SIZE":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.MeteringChannelSize = n
		}
	case "METERING_DEDUP_RETENTION_MS":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.MeteringDedupRetentionMs = n
		}
	case "METERING_BATCH_SIZE":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.MeteringBatchSize = n
		}
	case "METERING_BATCH_FLUSH_MS":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.MeteringBatchFlushMs = n
		}
	}
}
