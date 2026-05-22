package broker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

// ProxyRequestEvent describes one proxied request outcome.
type ProxyRequestEvent struct {
	TenantID      string `json:"tenant_id"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	UpstreamBase  string `json:"upstream_base"`
	StatusCode    int    `json:"status_code"`
	BytesSent     int64  `json:"bytes_sent"`
	BytesReceived int64  `json:"bytes_received"`
	LatencyMS     int64  `json:"latency_ms"`
	Error         string `json:"error,omitempty"`
}

func appendProxyEventAsync(store eventstore.EventStore, logger *log.Logger, ev ProxyRequestEvent) {
	if store == nil {
		return
	}
	go func() {
		b, err := json.Marshal(ev)
		if err != nil {
			return
		}
		_, err = store.Append(context.Background(), eventstore.Event{ID: time.Now().UTC().Format(time.RFC3339Nano), Type: "proxy_request", OccurredAt: time.Now().UTC(), PartitionKey: ev.TenantID, Source: "broker", Data: b})
		if err != nil && logger != nil {
			logger.Printf("event append failed: %v", err)
		}
	}()
}
