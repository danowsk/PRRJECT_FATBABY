package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

var (
	errInvalidID         = errors.New("envelope id is required")
	errInvalidType       = errors.New("envelope type is required")
	errInvalidSource     = errors.New("envelope source is required")
	errInvalidVersion    = errors.New("envelope version must be >= 1")
	errInvalidOccurredAt = errors.New("envelope occurred_at is required")
)

// NewEnvelope builds a fully populated eventstore.Event ready to append.
// ticker and sourceID may be empty for event types that don't apply.
func NewEnvelope(
	eventType string,
	version int,
	source string,
	sourceID string,
	ticker string,
	occurredAt time.Time,
	payload any,
) (eventstore.Event, error) {
	if version <= 0 {
		version = 1
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return eventstore.Event{}, fmt.Errorf("marshal envelope payload: %w", err)
	}

	id := ""
	if sourceID != "" {
		id = eventType + ":" + sourceID
	} else {
		id, err = uuidV4()
		if err != nil {
			return eventstore.Event{}, fmt.Errorf("generate event id: %w", err)
		}
	}

	ev := eventstore.Event{
		ID:         id,
		Type:       eventType,
		Source:     source,
		OccurredAt: occurredAt.UTC(),
		Data:       raw,
		Version:    version,
		Ticker:     ticker,
		SourceID:   sourceID,
	}

	if err := ValidateEnvelope(ev); err != nil {
		return eventstore.Event{}, err
	}

	return ev, nil
}

// ParseEnvelope decodes a raw JSON record into an eventstore.Event.
func ParseEnvelope(data []byte) (eventstore.Event, error) {
	var ev eventstore.Event
	if err := json.Unmarshal(data, &ev); err != nil {
		return eventstore.Event{}, err
	}
	return ev, nil
}

// ValidateEnvelope returns an error if required fields are missing or invalid.
// Required: ID, Type, Source, Version >= 1, OccurredAt non-zero.
func ValidateEnvelope(ev eventstore.Event) error {
	if ev.ID == "" {
		return errInvalidID
	}
	if ev.Type == "" {
		return errInvalidType
	}
	if ev.Source == "" {
		return errInvalidSource
	}
	if ev.Version < 1 {
		return errInvalidVersion
	}
	if ev.OccurredAt.IsZero() {
		return errInvalidOccurredAt
	}
	return nil
}

func uuidV4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	buf := make([]byte, 36)
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf), nil
}
