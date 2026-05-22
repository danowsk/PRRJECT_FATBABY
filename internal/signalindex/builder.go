package signalindex

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

// Build scans the entire store from sequence 1 and populates idx.
func Build(ctx context.Context, store eventstore.EventStore, idx *Index, logger *log.Logger) error {
	fromSeq := uint64(1)
	scanned := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		recs, err := store.ReadFrom(ctx, fromSeq, 512)
		if err != nil {
			return err
		}
		if len(recs) == 0 {
			return nil
		}
		for _, rec := range recs {
			if err := idx.Ingest(rec); err != nil {
				return err
			}
			scanned++
			if scanned%1000 == 0 && logger != nil {
				logger.Printf("signalindex build scanned=%d latest_seq=%d", scanned, rec.Sequence)
			}
			fromSeq = rec.Sequence + 1
		}
	}
}

// Tail starts a goroutine that polls the store every pollInterval for new records.
func Tail(ctx context.Context, store eventstore.EventStore, idx *Index, pollInterval time.Duration, logger *log.Logger) (ready <-chan struct{}) {
	readyCh := make(chan struct{})
	go func() {
		defer close(readyCh)
		t := time.NewTicker(pollInterval)
		defer t.Stop()
		readySent := false
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				start := idx.LatestSeq() + 1
				recs, err := store.ReadFrom(ctx, start, 256)
				if err != nil {
					if logger != nil && !errors.Is(err, context.Canceled) {
						logger.Printf("signalindex tail read error: %v", err)
					}
				} else {
					for _, rec := range recs {
						if err := idx.Ingest(rec); err != nil {
							if logger != nil {
								logger.Printf("signalindex tail ingest error: %v", err)
							}
						}
					}
				}
				if !readySent {
					readySent = true
					readyCh <- struct{}{}
				}
			}
		}
	}()
	return readyCh
}
