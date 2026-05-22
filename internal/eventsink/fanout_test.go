package eventsink

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

type stubSink struct { fn func(context.Context, eventstore.Event) error }
func (s stubSink) Write(ctx context.Context, evt eventstore.Event) error { return s.fn(ctx, evt) }

func TestFanoutPrimaryFailureStopsSecondary(t *testing.T) {
	var secCalls atomic.Int32
	f := &FanoutSink{Primary: stubSink{fn: func(context.Context, eventstore.Event) error { return errors.New("boom") }}, Secondary: []EventSink{stubSink{fn: func(context.Context, eventstore.Event) error { secCalls.Add(1); return nil }}}}
	err := f.Write(context.Background(), eventstore.Event{ID:"1",Type:"x",Data:[]byte(`{}`)})
	if err == nil { t.Fatal("expected error") }
	if secCalls.Load() != 0 { t.Fatalf("secondary called") }
}

func TestFanoutSecondaryAsync(t *testing.T) {
	done := make(chan struct{},1)
	f := &FanoutSink{Primary: stubSink{fn: func(context.Context, eventstore.Event) error { return nil }}, Secondary: []EventSink{stubSink{fn: func(context.Context, eventstore.Event) error { done<-struct{}{}; return nil }}}, SecondaryWorkers:1}
	if err := f.Write(context.Background(), eventstore.Event{ID:"1",Type:"x",Data:[]byte(`{}`)}); err != nil { t.Fatal(err) }
	select { case <-done: case <-time.After(time.Second): t.Fatal("timeout") }
}
