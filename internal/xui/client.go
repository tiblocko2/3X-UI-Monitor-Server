// Package xui предоставляет клиент для API панели 3X-UI.
package xui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"vpn-monitor/internal/config"
)

// Client — HTTP-клиент к 3X-UI с поддержкой сессий через cookie.
type Client struct {
	cfg        config.XUI
	httpClient *http.Client
}

// New создаёт Client с таймаутом 5 секунд.
func New(cfg config.XUI) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Jar:     newCookieJar(),
		},
	}
}

// OnlineCount возвращает количество онлайн-клиентов или -1 при ошибке.
func (c *Client) OnlineCount() int {
	if err := c.login(); err != nil {
		log.Printf("[xui] login failed: %v", err)
		return -1
	}

	count, err := c.fetchOnline()
	if err != nil {
		log.Printf("[xui] fetchOnline failed: %v", err)
		return -1
	}

	return count
}

func (c *Client) login() error {
	body := strings.NewReader(
		"username=" + url.QueryEscape(c.cfg.Username) +
			"&password=" + url.QueryEscape(c.cfg.Password),
	)

	req, err := http.NewRequest(http.MethodPost, c.cfg.BaseURL+"/login", body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// 3X-UI всегда отвечает HTTP 200; успех определяется по телу ответа.
	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("server rejected login: %s", result.Msg)
	}

	return nil
}

func (c *Client) fetchOnline() (int, error) {
	req, err := http.NewRequest(http.MethodPost, c.cfg.BaseURL+"/panel/api/inbounds/onlines", nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Obj []string `json:"obj"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	return len(result.Obj), nil
}

// ─── cookieJar ────────────────────────────────────────────────────────────────
// Минималистичная реализация http.CookieJar для хранения сессионной куки 3X-UI.

type cookieJar struct {
	mu      sync.Mutex
	cookies map[string][]*http.Cookie
}

func newCookieJar() *cookieJar {
	return &cookieJar{cookies: make(map[string][]*http.Cookie)}
}

func (j *cookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cookies[u.Host] = cookies
}

func (j *cookieJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.cookies[u.Host]
}
