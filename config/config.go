package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                      string
	ProxyAPIKey               string
	BaseURL                   string
	Accounts                  []string // DeepSeek web tokens, one per account
	AccountsFile              string
	DefaultModel              string
	ConversationTTL           time.Duration
	MaxConvs                  int
	AccountQPM                int // per-account requests/min (default 30)
	MaxConcurrentPerAccount   int // in-flight concurrency per account (default 5)
	RetryMaxAttempts          int // max 429 retry attempts (default 3)
	RetryBaseDelay            time.Duration
	RequestJitterMin          time.Duration // min random delay before each completion (default 200ms)
	RequestJitterMax          time.Duration // max random delay before each completion (default 800ms)
	PowJitterMin              time.Duration // min artificial delay when solving POW (default 30ms)
	PowJitterMax              time.Duration // max artificial delay when solving POW (default 150ms)
	SessionCreateJitter       time.Duration // random delay before creating a new session (default 200ms)
	DeviceIDRotateHours       int           // rotate deviceID every N hours (0 = disabled, default 0)
}

// Load reads configuration from the environment. Accounts come from
// DEEPSEEK_TOKEN plus one-per-line entries in DEEPSEEK_ACCOUNTS_FILE.
func Load() (*Config, error) {
	retryBase := env("RETRY_BASE_DELAY", "2s")
	d, err := time.ParseDuration(retryBase)
	if err != nil {
		return nil, fmt.Errorf("invalid RETRY_BASE_DELAY: %w", err)
	}
	cfg := &Config{
		Port:                      env("PORT", "8080"),
		ProxyAPIKey:               env("PROXY_API_KEY", ""),
		BaseURL:                   strings.TrimRight(env("DEEPSEEK_BASE_URL", "https://chat.deepseek.com"), "/"),
		AccountsFile:              env("DEEPSEEK_ACCOUNTS_FILE", "accounts.txt"),
		DefaultModel:              env("DEFAULT_MODEL", "deepseek-chat"),
		ConversationTTL:           30 * time.Minute,
		MaxConvs:                  1024,
		AccountQPM:                parseIntEnv("ACCOUNT_QPM", 6),
		MaxConcurrentPerAccount:   parseIntEnv("MAX_CONCURRENT_PER_ACCOUNT", 2),
		RetryMaxAttempts:          parseIntEnv("RETRY_MAX_ATTEMPTS", 3),
		RetryBaseDelay:            d,
		RequestJitterMin:          parseDurationEnv("REQUEST_JITTER_MIN", 200*time.Millisecond),
		RequestJitterMax:          parseDurationEnv("REQUEST_JITTER_MAX", 800*time.Millisecond),
		PowJitterMin:              parseDurationEnv("POW_JITTER_MIN", 30*time.Millisecond),
		PowJitterMax:              parseDurationEnv("POW_JITTER_MAX", 150*time.Millisecond),
		SessionCreateJitter:       parseDurationEnv("SESSION_CREATE_JITTER", 200*time.Millisecond),
		DeviceIDRotateHours:       parseIntEnv("DEVICE_ID_ROTATE_HOURS", 0),
	}
	if cfg.ProxyAPIKey == "" {
		return nil, fmt.Errorf("PROXY_API_KEY is required")
	}
	if v := os.Getenv("CONVERSATION_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid CONVERSATION_TTL: %w", err)
		}
		cfg.ConversationTTL = d
	}
	if v := os.Getenv("MAX_CONVERSATIONS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("invalid MAX_CONVERSATIONS: %q", v)
		}
		cfg.MaxConvs = n
	}
	if t := strings.TrimSpace(os.Getenv("DEEPSEEK_TOKEN")); t != "" {
		cfg.Accounts = append(cfg.Accounts, t)
	}
	fileAccounts, err := loadAccountsFile(cfg.AccountsFile)
	if err != nil {
		return nil, err
	}
	cfg.Accounts = append(cfg.Accounts, fileAccounts...)
	if len(cfg.Accounts) == 0 {
		return nil, fmt.Errorf("no DeepSeek account configured; set DEEPSEEK_TOKEN or %s", cfg.AccountsFile)
	}
	return cfg, nil
}

func loadAccountsFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func parseIntEnv(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func parseDurationEnv(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= 0 {
			return d
		}
	}
	return def
}
