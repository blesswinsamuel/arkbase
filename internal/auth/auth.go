package auth

import (
	"crypto/subtle"
	"net/http"

	"github.com/blesswinsamuel/arkbase/internal/config"
)

// Middleware returns an HTTP middleware that enforces Basic Auth if enabled in config.
func Middleware(cfg config.AuthConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			user, pass, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="arkbase"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(cfg.Username)) == 1
			passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(cfg.Password)) == 1

			if !userMatch || !passMatch {
				w.Header().Set("WWW-Authenticate", `Basic realm="arkbase"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
