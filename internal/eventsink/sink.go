package eventsink

import (
	"context"

	"github.com/example/prrject-fatbaby/eventstore"
)

// EventSink defines the standard block interface for any event publication target.
type EventSink interface {
	Write(ctx context.Context, evt eventstore.Event) error
}
