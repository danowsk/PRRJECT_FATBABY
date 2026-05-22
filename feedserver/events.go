package feedserver

import (
	"context"
	"encoding/json"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

type FeedSessionEvent struct {
	SessionID      string `json:"session_id"`
	TenantID       string `json:"tenant_id"`
	FromSeq        uint64 `json:"from_seq"`
	LastAckedSeq   uint64 `json:"last_acked_seq"`
	FramesSent     int64  `json:"frames_sent"`
	BytesSent      int64  `json:"bytes_sent"`
	DurationMS     int64  `json:"duration_ms"`
	DisconnectCode string `json:"disconnect_code"`
}

func appendFeedSessionEvent(store eventstore.EventStore, ev FeedSessionEvent) {
	go func() {
		b, _ := json.Marshal(ev)
		_, _ = store.Append(context.Background(), eventstore.Event{ID: ev.SessionID + "-feed", Type: "feed_session", Data: b, OccurredAt: time.Now().UTC(), Source: "feedserver"})
	}()
}
