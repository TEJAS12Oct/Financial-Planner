package security

import (
	envconfig "FinancialPlannerGo/config/env_config"
	"errors"
	"strings"
	"time"
)

type Config struct {
	Enabled                 bool
	APIKey                  string
	IPWhitelistEnabled      bool
	AllowedIPs              []string
	ReportSigningSecret     string
	ReportLinkTTL           time.Duration
	RateLimitEnabled        bool
	ReportRequestsPerMinute int
	ReportRateLimitBurst    int
}

func LoadConfig() (Config, error) {
	if envconfig.AppConfig == nil {
		return Config{}, errors.New("application configuration is not initialized")
	}

	securityConfig := envconfig.AppConfig.Security
	rateLimitConfig := envconfig.AppConfig.RateLimit

	cfg := Config{
		Enabled:                 securityConfig.Enabled,
		APIKey:                  strings.TrimSpace(securityConfig.APIKey),
		IPWhitelistEnabled:      securityConfig.IPWhitelistEnabled,
		AllowedIPs:              splitCSV(securityConfig.IPWhitelist),
		ReportSigningSecret:     strings.TrimSpace(securityConfig.ReportSigningSecret),
		ReportLinkTTL:           time.Duration(securityConfig.ReportLinkTTLSeconds) * time.Second,
		RateLimitEnabled:        rateLimitConfig.Enabled,
		ReportRequestsPerMinute: rateLimitConfig.RequestsPerMinute,
		ReportRateLimitBurst:    rateLimitConfig.Burst,
	}

	if !cfg.Enabled {
		return cfg, nil
	}
	if cfg.APIKey == "" {
		return Config{}, errors.New("API_KEY is required when SECURITY_ENABLED=true")
	}
	if cfg.ReportSigningSecret == "" {
		return Config{}, errors.New("REPORT_SIGNING_SECRET is required when SECURITY_ENABLED=true")
	}
	if cfg.ReportLinkTTL <= 0 {
		return Config{}, errors.New("REPORT_LINK_TTL_SECONDS must be greater than 0")
	}
	if cfg.IPWhitelistEnabled && len(cfg.AllowedIPs) == 0 {
		return Config{}, errors.New("IP_WHITELIST must contain at least one IP or CIDR")
	}
	if cfg.RateLimitEnabled && (cfg.ReportRequestsPerMinute <= 0 || cfg.ReportRateLimitBurst <= 0) {
		return Config{}, errors.New("report rate-limit values must be greater than 0")
	}

	return cfg, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
