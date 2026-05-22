package broker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testRoute(u string) *Route {
	return &Route{TenantID: "t1", TenantKey: "tk", UpstreamBase: u, UpstreamKey: "up", Enabled: true}
}

func TestProxy_AuthRewrite(t *testing.T) {
	got := ""
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r.Header.Get("Authorization"); w.WriteHeader(204) }))
	defer up.Close()
	p := &ProxyHandler{Client: up.Client()}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req = req.WithContext(context.WithValue(req.Context(), routeContextKey{}, testRoute(up.URL)))
	p.ServeHTTP(rr, req)
	if got != "Bearer up" {
		t.Fatalf("got %q", got)
	}
}

func TestProxy_AllowedPathsEnforced(t *testing.T) {
	reg := &Registry{}
	m := map[string]*Route{"tk": {TenantID: "t", TenantKey: "tk", AllowedPaths: []string{"/v1/"}, Enabled: true}}
	reg.ptr.Store(&m)
	h := AuthMiddleware(reg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v2/x", nil)
	req.Header.Set("Authorization", "Bearer tk")
	h.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestRegistry_HotReload(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "routes.json")
	_ = os.WriteFile(p, []byte(`{"routes":[{"tenant_id":"a","tenant_key":"k1","upstream_base":"http://x","enabled":true}]}`), 0o644)
	r, _ := LoadRegistry(p)
	if _, ok := r.Resolve("k1"); !ok {
		t.Fatal("missing k1")
	}
	_ = os.WriteFile(p, []byte(`{"routes":[{"tenant_id":"a","tenant_key":"k2","upstream_base":"http://x","enabled":true}]}`), 0o644)
	_ = r.Reload()
	if _, ok := r.Resolve("k1"); ok {
		t.Fatal("k1 still present")
	}
	if _, ok := r.Resolve("k2"); !ok {
		t.Fatal("k2 missing")
	}
}

func TestProxy_StreamingPassthrough(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := w.(http.Flusher)
		for i := 0; i < 3; i++ {
			_, _ = io.WriteString(w, "chunk\n")
			f.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer up.Close()
	p := &ProxyHandler{Client: up.Client()}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req = req.WithContext(context.WithValue(req.Context(), routeContextKey{}, testRoute(up.URL)))
	p.ServeHTTP(rr, req)
	if strings.Count(rr.Body.String(), "chunk") != 3 {
		t.Fatal("missing chunks")
	}
}
