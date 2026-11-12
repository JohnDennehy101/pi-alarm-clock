package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func authMiddleware(next http.Handler, apiToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiToken == "" {
			writeError(
				w,
				http.StatusServiceUnavailable,
				"API authentication is not configured",
			)
			return
		}

		const prefix = "Bearer "

		header := r.Header.Get("Authorization")

		if !strings.HasPrefix(header, prefix) {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		providedToken := strings.TrimPrefix(header, prefix)

		if subtle.ConstantTimeCompare(
			[]byte(providedToken),
			[]byte(apiToken),
		) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
