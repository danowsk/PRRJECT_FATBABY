package eventsink

import (
	"context"
	"log"
	"sync"

	"github.com/example/prrject-fatbaby/eventstore"
)

type DeadLetterHandler func(ctx context.Context, sinkName string, evt eventstore.Event, err error)

type FanoutSink struct {
	Primary            EventSink
	Secondary          []EventSink
	SecondaryWorkers   int
	DeadLetter         DeadLetterHandler
	SecondaryQueueSize int

	jobs chan fanoutJob
	once sync.Once
}

type fanoutJob struct {
	sink EventSink
	evt  eventstore.Event
}

func (f *FanoutSink) Write(ctx context.Context, evt eventstore.Event) error {
	if f.Primary == nil {
		return ErrNilPrimarySink
	}
	if err := f.Primary.Write(ctx, evt); err != nil {
		return err
	}
	f.once.Do(f.startWorkers)
	for _, sink := range f.Secondary {
		if sink == nil {
			continue
		}
		select {
		case f.jobs <- fanoutJob{sink: sink, evt: evt}:
		default:
			f.deadLetter(ctx, "secondary_queue", evt, ErrSecondaryQueueFull)
		}
	}
	return nil
}

func (f *FanoutSink) startWorkers() {
	if f.SecondaryWorkers <= 0 {
		f.SecondaryWorkers = 4
	}
	if f.SecondaryQueueSize <= 0 {
		f.SecondaryQueueSize = 1024
	}
	f.jobs = make(chan fanoutJob, f.SecondaryQueueSize)
	for i := 0; i < f.SecondaryWorkers; i++ {
		go func() {
			for j := range f.jobs {
				if err := j.sink.Write(context.Background(), j.evt); err != nil {
					f.deadLetter(context.Background(), "secondary_sink", j.evt, err)
				}
			}
		}()
	}
}

func (f *FanoutSink) deadLetter(ctx context.Context, sinkName string, evt eventstore.Event, err error) {
	if f.DeadLetter != nil {
		f.DeadLetter(ctx, sinkName, evt, err)
		return
	}
	log.Printf("fanout deadletter sink=%s event_id=%s type=%s err=%v", sinkName, evt.ID, evt.Type, err)
}
