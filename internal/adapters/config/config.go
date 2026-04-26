package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken         string
	PostgresDSN      string
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	LogChatID        int64
	AdminIDs         map[int64]struct{}
	TelegramProxyURL string
}

func Load() (Config, error) {
	cfg := Config{
		BotToken:         strings.TrimSpace(os.Getenv("BOT_TOKEN")),
		PostgresDSN:      strings.TrimSpace(os.Getenv("POSTGRES_DSN")),
		RedisAddr:        valueOrDefault("REDIS_ADDR", "redis:6379"),
		RedisPassword:    strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		AdminIDs:         parseAdminIDs(os.Getenv("ADMIN_IDS")),
		TelegramProxyURL: strings.TrimSpace(os.Getenv("TELEGRAM_PROXY_URL")),
	}

	redisDBRaw := strings.TrimSpace(os.Getenv("REDIS_DB"))
	if redisDBRaw == "" {
		cfg.RedisDB = 0
	} else {
		v, err := strconv.Atoi(redisDBRaw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid REDIS_DB: %w", err)
		}
		cfg.RedisDB = v
	}
	logChatRaw := strings.TrimSpace(os.Getenv("LOG_CHAT_ID"))
	if logChatRaw == "" {
		cfg.LogChatID = 0
	} else {
		v, err := strconv.ParseInt(logChatRaw, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("invalid LOG_CHAT_ID: %w", err)
		}
		cfg.LogChatID = v
	}

	if cfg.BotToken == "" {
		return Config{}, fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.PostgresDSN == "" {
		return Config{}, fmt.Errorf("POSTGRES_DSN is required")
	}
	if cfg.TelegramProxyURL != "" {
		parsedURL, err := url.Parse(cfg.TelegramProxyURL)
		if err != nil {
			return Config{}, fmt.Errorf("invalid TELEGRAM_PROXY_URL: %w", err)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return Config{}, fmt.Errorf("invalid TELEGRAM_PROXY_URL: scheme and host are required")
		}
	}

	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func parseAdminIDs(raw string) map[int64]struct{} {
	res := make(map[int64]struct{})
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		v, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			continue
		}
		res[v] = struct{}{}
	}
	return res
}
