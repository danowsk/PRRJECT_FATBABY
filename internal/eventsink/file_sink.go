package eventsink

import (
	"context"

	"github.com/example/prrject-fatbaby/eventstore"
)

type FileSink struct { Store eventstore.EventStore }

func (f FileSink) Write(ctx context.Context, evt eventstore.Event) error {
	if _, err := f.Store.Append(ctx, evt); err != nil { return err }
	return nil
}
