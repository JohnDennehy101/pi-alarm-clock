package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/auth"
)

type deviceIDKey struct{}

func authMiddleware(next http.Handler, store *auth.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "

		header := r.Header.Get("Authorization")

		if !strings.HasPrefix(header, prefix) {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		providedToken := strings.TrimPrefix(header, prefix)

		device, ok := store.Verify(providedToken)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		ctx := context.WithValue(r.Context(), deviceIDKey{}, device.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func deviceIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(deviceIDKey{}).(string)
	return id
}
