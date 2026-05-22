package eventsink

import (
	"testing"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

func TestBuildS3ObjectKey(t *testing.T) {
	tm := time.Date(2026, 5, 2, 7, 4, 0, 0, time.UTC)
	e := eventstore.Event{ID: "abc-123", Source: "PRNewsWire", OccurredAt: tm}
	got := BuildS3ObjectKey(e)
	want := "source=prnewswire/year=2026/month=05/day=02/hour=07/abc-123.json.zst"
	if got != want { t.Fatalf("got %s want %s", got, want) }
}

func TestBuildS3ObjectKeyFallsBackToIngested(t *testing.T) {
	tm := time.Date(2025, 11, 9, 1, 0, 0, 0, time.UTC)
	e := eventstore.Event{ID: "evt", Source: "sec/watch", IngestedAt: tm}
	got := BuildS3ObjectKey(e)
	want := "source=sec_watch/year=2025/month=11/day=09/hour=01/evt.json.zst"
	if got != want { t.Fatalf("got %s want %s", got, want) }
}
