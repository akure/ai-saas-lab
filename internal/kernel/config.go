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

	for _, key := range []string{"HTTP_PORT", "LOG_LEVEL", "DAILY_TOKEN_QUOTA", "DATA_DIR", "LOCAL_TEST"} {
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
	}
}
