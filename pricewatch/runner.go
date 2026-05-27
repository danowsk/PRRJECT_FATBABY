package pricewatch

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
	"github.com/example/prrject-fatbaby/internal/events"
	"github.com/example/prrject-fatbaby/secwatch"
)

type stdLogger struct{}

func (stdLogger) Printf(format string, args ...any) { fmt.Printf(format+"\n", args...) }

func RunFetch(ctx context.Context, cfg RunnerConfig) (Summary, error) {
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.Logger == nil {
		cfg.Logger = stdLogger{}
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 3
	}
	if cfg.LookbackDays <= 0 {
		cfg.LookbackDays = 7
	}
	if cfg.Client == nil {
		cfg.Client = NewClient(ClientConfig{})
	}
	w, err := secwatch.LoadWatchlist(cfg.WatchlistPath)
	if err != nil {
		return Summary{}, err
	}
	entries := w.EnabledEntries()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Ticker < entries[j].Ticker })
	store, err := eventstore.NewFileStore(cfg.StoreRoot)
	if err != nil {
		return Summary{}, fmt.Errorf("open event store: %w", err)
	}
	defer store.Close()
	seen, err := loadSeenSourceIDs(ctx, store)
	if err != nil {
		return Summary{}, err
	}
	type result struct {
		ticker  string
		candles []DailyCandle
		err     error
	}
	jobs := make(chan secwatch.WatchEntry)
	results := make(chan result)
	for i := 0; i < cfg.Concurrency; i++ {
		go func() {
			for e := range jobs {
				c, err := cfg.Client.FetchDailyCandles(ctx, e.Ticker, cfg.LookbackDays)
				if err != nil {
					results <- result{ticker: e.Ticker, err: err}
					continue
				}
				results <- result{ticker: e.Ticker, candles: c}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, e := range entries {
			select {
			case <-ctx.Done():
				return
			case jobs <- e:
			}
		}
	}()
	s := Summary{Tickers: len(entries)}
	for i := 0; i < len(entries); i++ {
		r := <-results
		if r.err != nil {
			s.Failed++
			cfg.Logger.Printf("pricewatch ticker failed ticker=%s err=%v", r.ticker, r.err)
			continue
		}
		s.Fetched += len(r.candles)
		for _, c := range r.candles {
			sourceID := c.Ticker + ":" + c.Date.Format("2006-01-02")
			if _, ok := seen[sourceID]; ok {
				s.Skipped++
				continue
			}
			if cfg.DryRun {
				s.Persisted++
				cfg.Logger.Printf("pricewatch discovered ticker=%s date=%s open=%.4f high=%.4f low=%.4f close=%.4f adj_close=%.4f volume=%d", c.Ticker, c.Date.Format("2006-01-02"), c.Open, c.High, c.Low, c.Close, c.AdjClose, c.Volume)
				continue
			}
			ev, err := events.NewEnvelope(events.EventTypePriceCandleDaily, 1, "yahoo_finance", sourceID, c.Ticker, c.Date, events.PriceCandleDaily{Open: c.Open, High: c.High, Low: c.Low, Close: c.Close, AdjClose: c.AdjClose, Volume: c.Volume})
			if err != nil {
				s.Failed++
				continue
			}
			if _, err := store.Append(ctx, ev); err != nil {
				return s, fmt.Errorf("append candle %s: %w", sourceID, err)
			}
			seen[sourceID] = struct{}{}
			s.Persisted++
			cfg.Logger.Printf("pricewatch discovered ticker=%s date=%s open=%.4f high=%.4f low=%.4f close=%.4f adj_close=%.4f volume=%d", c.Ticker, c.Date.Format("2006-01-02"), c.Open, c.High, c.Low, c.Close, c.AdjClose, c.Volume)
		}
	}
	cfg.Logger.Printf("pricewatch summary tickers=%d fetched=%d persisted=%d skipped=%d failed=%d dry_run=%t", s.Tickers, s.Fetched, s.Persisted, s.Skipped, s.Failed, cfg.DryRun)
	return s, nil
}

func loadSeenSourceIDs(ctx context.Context, store eventstore.EventStore) (map[string]struct{}, error) {
	seen := map[string]struct{}{}
	from := uint64(1)
	for {
		recs, err := store.ReadFrom(ctx, from, 512)
		if err != nil {
			return nil, fmt.Errorf("read events for dedupe: %w", err)
		}
		if len(recs) == 0 {
			return seen, nil
		}
		for _, rec := range recs {
			if rec.Event.Type == events.EventTypePriceCandleDaily && rec.Event.SourceID != "" {
				seen[rec.Event.SourceID] = struct{}{}
			}
		}
		from = recs[len(recs)-1].Sequence + 1
	}
}
