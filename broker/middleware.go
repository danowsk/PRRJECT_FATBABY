package broker

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type routeContextKey struct{}

// AuthMiddleware authenticates inbound tenant requests and attaches route metadata.
func AuthMiddleware(reg *Registry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		route, ok := reg.Resolve(token)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unknown tenant")
			return
		}
		if len(route.AllowedPaths) > 0 {
			allowed := false
			for _, p := range route.AllowedPaths {
				if strings.HasPrefix(r.URL.Path, p) {
					allowed = true
					break
				}
			}
			if !allowed {
				writeJSONError(w, http.StatusForbidden, "path not allowed")
				return
			}
		}
		ctx := context.WithValue(r.Context(), routeContextKey{}, route)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RouteFromContext extracts route from context.
func RouteFromContext(ctx context.Context) (*Route, bool) {
	r, ok := ctx.Value(routeContextKey{}).(*Route)
	return r, ok
}

func bearerToken(v string) string {
	parts := strings.SplitN(v, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
