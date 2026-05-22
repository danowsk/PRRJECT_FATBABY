package newssite

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

// Handler is an http.Handler for the news site.
type Handler struct {
	store        eventstore.EventStore
	logger       *log.Logger
	defaultLimit int
}

// NewHandler returns a new Handler.
func NewHandler(store eventstore.EventStore, logger *log.Logger) *Handler {
	return &Handler{store: store, logger: logger, defaultLimit: 50}
}

// ServeHTTP dispatches list and detail routes for source documents.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	status := http.StatusOK
	defer func() {
		h.logger.Printf("newssite method=%s path=%s status=%d dur=%s", r.Method, r.URL.Path, status, time.Since(start))
	}()

	if r.Method != http.MethodGet {
		status = http.StatusNotFound
		http.NotFound(w, r)
		return
	}

	switch {
	case r.URL.Path == "/":
		entries, err := ReadLatest(r.Context(), h.store, h.defaultLimit)
		if err != nil {
			status = http.StatusInternalServerError
			http.Error(w, fmt.Sprintf("internal error: %v", err), status)
			return
		}
		var buf bytes.Buffer
		RenderListPage(&buf, entries)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	case strings.HasPrefix(r.URL.Path, "/doc/"):
		id := strings.TrimPrefix(r.URL.Path, "/doc/")
		entry, found, err := ReadByIdentity(r.Context(), h.store, id)
		if err != nil {
			status = http.StatusInternalServerError
			http.Error(w, fmt.Sprintf("internal error: %v", err), status)
			return
		}
		if !found {
			status = http.StatusNotFound
			http.NotFound(w, r)
			return
		}
		var buf bytes.Buffer
		RenderDetailPage(&buf, entry)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	default:
		status = http.StatusNotFound
		http.NotFound(w, r)
	}
}
