package broker

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

var hopByHopHeaders = map[string]struct{}{"Connection": {}, "Keep-Alive": {}, "Proxy-Authenticate": {}, "Proxy-Authorization": {}, "Te": {}, "Trailer": {}, "Transfer-Encoding": {}, "Upgrade": {}}

type countWriter struct {
	n *int64
	w io.Writer
}

func (c countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	atomic.AddInt64(c.n, int64(n))
	return n, err
}

type flushWriter struct {
	w         http.ResponseWriter
	n         int
	threshold int
}

func (f *flushWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	f.n += n
	if f.n >= f.threshold {
		f.n = 0
		if fl, ok := f.w.(http.Flusher); ok {
			fl.Flush()
		}
	}
	return n, err
}

// ProxyHandler proxies requests to tenant upstreams.
type ProxyHandler struct {
	Client        *http.Client
	Store         eventstore.EventStore
	Logger        *log.Logger
	FlushBytes    int
	RequestsTotal atomic.Int64
	ErrorsTotal   atomic.Int64
	BytesIn       atomic.Int64
	BytesOut      atomic.Int64
}

// ServeHTTP proxies the request.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.RequestsTotal.Add(1)
	route, ok := RouteFromContext(r.Context())
	if !ok {
		h.ErrorsTotal.Add(1)
		writeJSONError(w, http.StatusUnauthorized, "unknown tenant")
		return
	}
	if isUpgradeRequest(r.Header) {
		h.handleUpgrade(w, r, route, start)
		return
	}
	status := http.StatusBadGateway
	var errText string
	var sent, recv int64
	u, err := url.Parse(route.UpstreamBase)
	if err != nil {
		h.ErrorsTotal.Add(1)
		writeJSONError(w, http.StatusBadGateway, "upstream unavailable")
		return
	}
	u.Path = strings.TrimRight(u.Path, "/") + r.URL.Path
	u.RawQuery = r.URL.RawQuery
	body := io.Reader(r.Body)
	limit := route.MaxBodyBytes
	if limit == 0 {
		limit = 32 << 20
	}
	body = http.MaxBytesReader(w, r.Body, limit)
	upReq, err := http.NewRequestWithContext(r.Context(), r.Method, u.String(), io.NopCloser(io.TeeReader(body, countWriter{n: &sent, w: io.Discard})))
	if err != nil {
		h.ErrorsTotal.Add(1)
		writeJSONError(w, http.StatusBadGateway, "upstream unavailable")
		return
	}
	copyHeadersSansHop(upReq.Header, r.Header)
	applyHeaderPolicy(upReq.Header, route)
	setForwardHeaders(upReq, r)
	upReq.Host = u.Host
	resp, err := h.Client.Do(upReq)
	if err != nil {
		h.ErrorsTotal.Add(1)
		writeJSONError(w, http.StatusBadGateway, "upstream unavailable")
		errText = err.Error()
		h.emit(route, r, status, sent, recv, start, errText)
		return
	}
	defer resp.Body.Close()
	status = resp.StatusCode
	copyHeadersSansHop(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	writer := io.Writer(w)
	if _, ok := w.(http.Flusher); ok {
		threshold := h.FlushBytes
		if threshold <= 0 {
			threshold = 4096
		}
		writer = &flushWriter{w: w, threshold: threshold}
	}
	_, err = io.Copy(countWriter{n: &recv, w: writer}, resp.Body)
	if err != nil {
		errText = err.Error()
		h.ErrorsTotal.Add(1)
	}
	for k, vals := range resp.Trailer {
		for _, v := range vals {
			w.Header().Add(http.TrailerPrefix+k, v)
		}
	}
	h.BytesIn.Add(sent)
	h.BytesOut.Add(recv)
	h.emit(route, r, status, sent, recv, start, errText)
}

func (h *ProxyHandler) emit(route *Route, r *http.Request, status int, sent, recv int64, start time.Time, err string) {
	lat := time.Since(start).Milliseconds()
	if h.Logger != nil {
		b, _ := json.Marshal(map[string]any{"tenant_id": route.TenantID, "method": r.Method, "path": r.URL.Path, "status": status, "latency_ms": lat, "bytes_sent": sent, "bytes_received": recv})
		h.Logger.Print(string(b))
	}
	appendProxyEventAsync(h.Store, h.Logger, ProxyRequestEvent{TenantID: route.TenantID, Method: r.Method, Path: r.URL.Path, UpstreamBase: route.UpstreamBase, StatusCode: status, BytesSent: sent, BytesReceived: recv, LatencyMS: lat, Error: err})
}

func isUpgradeRequest(h http.Header) bool {
	return strings.EqualFold(h.Get("Connection"), "Upgrade") || h.Get("Upgrade") != ""
}
func copyHeadersSansHop(dst, src http.Header) {
	for k, vals := range src {
		if _, drop := hopByHopHeaders[http.CanonicalHeaderKey(k)]; drop {
			continue
		}
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}
func applyHeaderPolicy(h http.Header, r *Route) {
	for _, k := range r.StripHeaders {
		h.Del(k)
	}
	for k, v := range r.InjectHeaders {
		h.Set(k, v)
	}
	h.Set("Authorization", "Bearer "+r.UpstreamKey)
}
func setForwardHeaders(upReq, in *http.Request) {
	host, _, _ := net.SplitHostPort(in.RemoteAddr)
	if host == "" {
		host = in.RemoteAddr
	}
	prior := in.Header.Get("X-Forwarded-For")
	if prior != "" {
		host = prior + ", " + host
	}
	upReq.Header.Set("X-Forwarded-For", host)
	upReq.Header.Set("X-Forwarded-Host", in.Host)
	if in.TLS != nil {
		upReq.Header.Set("X-Forwarded-Proto", "https")
	} else {
		upReq.Header.Set("X-Forwarded-Proto", "http")
	}
}

func (h *ProxyHandler) handleUpgrade(w http.ResponseWriter, r *http.Request, route *Route, start time.Time) {
	writeJSONError(w, http.StatusNotImplemented, "upgrade not supported")
	h.emit(route, r, http.StatusNotImplemented, 0, 0, start, "")
}

// HealthHandler returns health status.
func HealthHandler(reg *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "routes": reg.Count()})
	}
}

// MetricsHandler returns plaintext metrics.
func (h *ProxyHandler) MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = io.WriteString(w, "broker_requests_total "+itoa(h.RequestsTotal.Load())+"\n")
		_, _ = io.WriteString(w, "broker_errors_total "+itoa(h.ErrorsTotal.Load())+"\n")
		_, _ = io.WriteString(w, "broker_bytes_in_total "+itoa(h.BytesIn.Load())+"\n")
		_, _ = io.WriteString(w, "broker_bytes_out_total "+itoa(h.BytesOut.Load())+"\n")
	}
}
func itoa(v int64) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(time.Duration(v).String(), "ns", ""), "s", ""))
}

var _ = context.Background
