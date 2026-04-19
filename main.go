package main

import (
	"embed"
	"io/fs"
	"log"

	"vpn-monitor/internal/collector"
	"vpn-monitor/internal/config"
	"vpn-monitor/internal/server"
	"vpn-monitor/internal/store"
	"vpn-monitor/internal/xui"
)

//go:embed web
var webFS embed.FS

func main() {
	cfg := config.Load()

	// Хранилище метрик.
	dataStore := store.New(cfg.Collector.MaxDataPoints)

	// Клиент к 3X-UI.
	xuiClient := xui.New(cfg.XUI)

	// Коллектор — запускаем в фоне.
	col := collector.New(cfg.Collector, dataStore, xuiClient)
	go col.Run()

	// Подфайловая система для веб-ресурсов (strip префикса "web/").
	webSubFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("failed to sub webFS: %v", err)
	}

	// HTTP-сервер — запускаем синхронно (блокирует до ошибки).
	srv := server.New(cfg.App, webSubFS, dataStore)
	log.Fatal(srv.Run())
}
