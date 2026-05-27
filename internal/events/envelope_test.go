package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

func TestNewEnvelopeProducesValidEnvelopes(t *testing.T) {
	t.Parallel()
	occurredAt := time.Date(2026, 5, 26, 12, 30, 0, 123, time.UTC)

	tests := []struct {
		name      string
		eventType string
		sourceID  string
		payload   any
	}{
		{name: "price_candle_daily", eventType: EventTypePriceCandleDaily, sourceID: "NVDA:2026-05-26", payload: PriceCandleDaily{Open: 1, High: 2, Low: 0.5, Close: 1.5, AdjClose: 1.4, Volume: 100}},
		{name: "fundamental_snapshot", eventType: EventTypeFundamentalSnapshot, sourceID: "NVDA:2026Q1", payload: FundamentalSnapshot{Revenue: 1_000_000, Period: "2026Q1", FiscalPeriodEnd: occurredAt.Format(time.RFC3339)}},
		{name: "signal_emitted", eventType: EventTypeSignalEmitted, sourceID: "", payload: map[string]any{"signal": "buy", "score": 0.92}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ev, err := NewEnvelope(tt.eventType, 0, "ingestor", tt.sourceID, "NVDA", occurredAt, tt.payload)
			if err != nil {
				t.Fatalf("NewEnvelope() error = %v", err)
			}
			if err := ValidateEnvelope(ev); err != nil {
				t.Fatalf("ValidateEnvelope() error = %v", err)
			}
			if ev.Version != 1 {
				t.Fatalf("Version = %d, want 1", ev.Version)
			}
			if ev.OccurredAt.Location() != time.UTC {
				t.Fatalf("OccurredAt location = %v, want UTC", ev.OccurredAt.Location())
			}
			if len(ev.Data) == 0 {
				t.Fatal("Data is empty")
			}
		})
	}
}

func TestValidateEnvelopeRejectsInvalid(t *testing.T) {
	t.Parallel()
	valid := eventstore.Event{ID: "id", Type: "type", Source: "source", Version: 1, OccurredAt: time.Now().UTC()}
	cases := []struct {
		name string
		ev   eventstore.Event
	}{
		{name: "missing id", ev: func() eventstore.Event { v := valid; v.ID = ""; return v }()},
		{name: "missing type", ev: func() eventstore.Event { v := valid; v.Type = ""; return v }()},
		{name: "missing source", ev: func() eventstore.Event { v := valid; v.Source = ""; return v }()},
		{name: "zero occurred_at", ev: func() eventstore.Event { v := valid; v.OccurredAt = time.Time{}; return v }()},
		{name: "version less than one", ev: func() eventstore.Event { v := valid; v.Version = 0; return v }()},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateEnvelope(tc.ev); err == nil {
				t.Fatal("ValidateEnvelope() error = nil, want non-nil")
			}
		})
	}
}

func TestParseEnvelopeRoundTrip(t *testing.T) {
	t.Parallel()
	input, err := NewEnvelope(EventTypePriceCandleDaily, 2, "ingestor", "NVDA:2026-05-26", "NVDA", time.Now().UTC(), PriceCandleDaily{Open: 1, High: 2, Low: 0.5, Close: 1.5, AdjClose: 1.4, Volume: 100})
	if err != nil {
		t.Fatalf("NewEnvelope() error = %v", err)
	}

	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	parsed, err := ParseEnvelope(raw)
	if err != nil {
		t.Fatalf("ParseEnvelope() error = %v", err)
	}

	if parsed.ID != input.ID || parsed.Type != input.Type || parsed.SourceID != input.SourceID || parsed.Ticker != input.Ticker || parsed.Version != input.Version {
		t.Fatalf("parsed envelope mismatch: got %+v want %+v", parsed, input)
	}
}

func TestNewEnvelopeIDGenerationFromSourceID(t *testing.T) {
	t.Parallel()
	ev, err := NewEnvelope(EventTypePriceCandleDaily, 1, "ingestor", "NVDA:2026-05-26", "NVDA", time.Now().UTC(), PriceCandleDaily{})
	if err != nil {
		t.Fatalf("NewEnvelope() error = %v", err)
	}
	want := EventTypePriceCandleDaily + ":NVDA:2026-05-26"
	if ev.ID != want {
		t.Fatalf("ID = %q, want %q", ev.ID, want)
	}
}
