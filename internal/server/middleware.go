package server

import (
	"net/http"

	"github.com/gorilla/sessions"
)

// authMiddleware проверяет сессию и редиректит на /login если пользователь не авторизован.
func authMiddleware(sessionStore *sessions.CookieStore, sessionName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Страница логина доступна без авторизации.
			if r.URL.Path == "/login" || r.URL.Path == "/login/" {
				next.ServeHTTP(w, r)
				return
			}

			sess, _ := sessionStore.Get(r, sessionName)
			if auth, ok := sess.Values["auth"].(bool); !ok || !auth {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
