package config

import (
	"os"
	"time"
)

// App — настройки веб-сервера мониторинга.
type App struct {
	Port          string
	Username      string
	Password      string
	SessionSecret string
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
	Interval      time.Duration
	MaxDataPoints int
}

// Config — корневая конфигурация приложения.
type Config struct {
	App       App
	XUI       XUI
	Collector Collector
}

// Load читает конфигурацию из переменных окружения с fallback на дефолты.
func Load() Config {
	return Config{
		App: App{
			Port:          env("PORT", "8080"),
			Username:      "Akvil0n",
			Password:      "Perfect10nizm",
			SessionSecret: env("SESSION_SECRET", "super-secret-key-change-me-pls"),
			CertFile:      env("TLS_CERT", "/root/cert/akvilon.nemesh-vpn.ru/fullchain.pem"),
			KeyFile:       env("TLS_KEY", "/root/cert/akvilon.nemesh-vpn.ru/privkey.pem"),
		},
		XUI: XUI{
			BaseURL:  env("XUI_URL", "https://akvilon.nemesh-vpn.ru:808"),
			Username: env("XUI_USER", "Akvil0n"),
			Password: env("XUI_PASS", "Perfect10nizm"),
		},
		Collector: Collector{
			Interval:      10 * time.Second,
			MaxDataPoints: 1440, // 24ч при интервале 60с
		},
	}
}

func env(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
