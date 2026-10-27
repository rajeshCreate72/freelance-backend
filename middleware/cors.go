package middleware

import (
	"net/http"
	"strings"
)

func CORS(originsList string, next http.Handler) http.Handler {
	allowedOrigins := make(map[string]bool)

	for _, origin := range strings.Split(originsList, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowedOrigins[origin] = true
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		isAllowed := allowedOrigins[origin]

		if isAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, PATCH, DELETE, OPTIONS",
			)
			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Content-Type, Authorization",
			)

			// Browser sends OPTIONS first to check if the real request is allowed
			if r.Method == http.MethodOptions {
				if !isAllowed {
					http.Error(w, "CORS origin not allowed", http.StatusForbidden)
					return
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}