package pricewatch

import (
	"context"
	"testing"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
	"github.com/example/prrject-fatbaby/internal/events"
)

func TestLoadSeenSourceIDs(t *testing.T) {
	dir := t.TempDir()
	store, err := eventstore.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ev1, _ := events.NewEnvelope(events.EventTypePriceCandleDaily, 1, "yahoo_finance", "AAPL:2026-05-26", "AAPL", time.Now().UTC(), events.PriceCandleDaily{Close: 1})
	ev2 := eventstore.Event{ID: "x", Type: "other", Source: "x", Data: []byte(`{"ok":true}`)}
	if _, err := store.Append(context.Background(), ev1, ev2); err != nil {
		t.Fatal(err)
	}
	seen, err := loadSeenSourceIDs(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := seen["AAPL:2026-05-26"]; !ok {
		t.Fatal("expected seen source id")
	}
	if _, ok := seen["missing"]; ok {
		t.Fatal("unexpected source id")
	}
}
