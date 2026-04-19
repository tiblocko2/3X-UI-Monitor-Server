package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// App — настройки веб-сервера мониторинга.
type App struct {
	Port          string
	Username      string
	Password      string
	SessionSecret string
	HTTPS         bool
	CertFile      string
	KeyFile       string
}

// XUI — настройки подключения к панели 3X-UI.
type XUI struct {
	BaseURL  string
	Username string
	Password string
}

// Collector — параметры сбора метрик.
type Collector struct {
	Interval time.Duration
}

// Config — корневая конфигурация приложения.
type Config struct {
	App           App
	XUI           XUI
	Collector     Collector
	DataDir       string
	RetentionDays int
}

// Load читает конфигурацию из переменных окружения и завершает процесс при ошибке.
func Load() Config {
	secret := env("SESSION_SECRET", "")
	if secret == "" {
		secret = generateSecret()
	}

	httpsEnabled := env("HTTPS_ENABLED", "false") == "true"

	retentionDays := 30
	if v := env("RETENTION_DAYS", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retentionDays = n
		}
	}

	cfg := Config{
		App: App{
			Port:          env("PORT", "8080"),
			Username:      requireEnv("DASH_USER"),
			Password:      requireEnv("DASH_PASS"),
			SessionSecret: secret,
			HTTPS:         httpsEnabled,
			CertFile:      env("TLS_CERT", ""),
			KeyFile:       env("TLS_KEY", ""),
		},
		XUI: XUI{
			BaseURL:  requireEnv("XUI_URL"),
			Username: requireEnv("XUI_USER"),
			Password: requireEnv("XUI_PASS"),
		},
		Collector: Collector{
			Interval: 10 * time.Second,
		},
		DataDir:       env("DATA_DIR", "/var/lib/vpn-monitor"),
		RetentionDays: retentionDays,
	}

	if httpsEnabled && (cfg.App.CertFile == "" || cfg.App.KeyFile == "") {
		log.Fatal("[config] HTTPS_ENABLED=true requires TLS_CERT and TLS_KEY to be set")
	}

	return cfg
}

func env(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[config] required environment variable %s is not set", key)
	}
	return v
}

func generateSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate session secret: %v", err))
	}
	return hex.EncodeToString(b)
}
