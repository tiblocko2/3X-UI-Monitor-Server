package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"

	"github.com/gorilla/sessions"

	"vpn-monitor/internal/store"
)

// handlers группирует зависимости для HTTP-хендлеров.
type handlers struct {
	webFS        fs.FS
	sessionStore *sessions.CookieStore
	sessionName  string
	appUsername  string
	appPassword  string
	dataStore    *store.Store
}

// handleLogin обрабатывает GET (показать форму) и POST (проверить данные).
func (h *handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.FormValue("username") == h.appUsername && r.FormValue("password") == h.appPassword {
			sess, _ := h.sessionStore.Get(r, h.sessionName)
			sess.Values["auth"] = true
			sess.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/login?err=1", http.StatusFound)
		return
	}
	serveFile(w, r, h.webFS, "login.html")
}

// handleLogout сбрасывает сессию и редиректит на /login.
func (h *handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.sessionStore.Get(r, h.sessionName)
	sess.Values["auth"] = false
	sess.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// handleIndex отдаёт главную страницу дашборда.
func (h *handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, h.webFS, "index.html")
}

// handleData отдаёт все накопленные DataPoint-ы в JSON.
func (h *handlers) handleData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.dataStore.GetAll())
}

// serveFile — замена http.ServeFileFS, которая появилась только в Go 1.22.
func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) {
	f, err := fsys.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// embed.FS возвращает файлы реализующие io.ReadSeeker, поэтому assert безопасен.
	type readSeekFile interface {
		fs.File
		io.ReadSeeker
	}
	http.ServeContent(w, r, name, stat.ModTime(), f.(readSeekFile))
}
