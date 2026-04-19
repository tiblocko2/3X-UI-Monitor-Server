// Package server собирает HTTP-сервер: роутер, middleware, TLS.
package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"vpn-monitor/internal/config"
	"vpn-monitor/internal/store"
)

// Server инкапсулирует HTTP-сервер и все его зависимости.
type Server struct {
	cfg     config.App
	router  *mux.Router
	handler handlers
}

// New собирает Server из зависимостей.
func New(cfg config.App, webFS fs.FS, dataStore *store.Store) *Server {
	sessionStore := sessions.NewCookieStore([]byte(cfg.SessionSecret))

	h := handlers{
		webFS:        webFS,
		sessionStore: sessionStore,
		sessionName:  "vpnmon",
		appUsername:  cfg.Username,
		appPassword:  cfg.Password,
		dataStore:    dataStore,
	}

	s := &Server{cfg: cfg, handler: h}
	s.router = s.buildRouter(sessionStore, webFS)
	return s
}

// Run запускает HTTPS-сервер. Блокирует до завершения.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%s", s.cfg.Port)
	log.Printf("[server] HTTPS listening on %s", addr)
	return http.ListenAndServe(addr, s.router)
	//return http.ListenAndServeTLS(addr, s.cfg.CertFile, s.cfg.KeyFile, s.router)
}

func (s *Server) buildRouter(sessionStore *sessions.CookieStore, webFS fs.FS) *mux.Router {
	r := mux.NewRouter()

	// Auth middleware применяется ко всем маршрутам.
	r.Use(authMiddleware(sessionStore, "vpnmon"))

	// Страницы и API.
	r.HandleFunc("/login", s.handler.handleLogin).Methods(http.MethodGet, http.MethodPost)
	r.HandleFunc("/logout", s.handler.handleLogout)
	r.HandleFunc("/api/data", s.handler.handleData)
	r.HandleFunc("/", s.handler.handleIndex)

	// Статические файлы из embed.FS.
	// http.FileServer(http.FS(...)) совместим с Go 1.21 в отличие от http.FileServerFS.
	staticFS, _ := fs.Sub(webFS, "static")
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))),
	)

	return r
}
