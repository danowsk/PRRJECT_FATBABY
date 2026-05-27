package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/example/prrject-fatbaby/pricewatch"
)

func main() {
	var watchlistPath = flag.String("watchlist", filepath.Join("config", "watchlist.json"), "watchlist config path")
	var storeRoot = flag.String("store", filepath.Join("var", "pricewatch"), "event store root")
	var dryRun = flag.Bool("dry-run", false, "fetch but do not persist")
	var lookback = flag.Int("lookback", 7, "days of history to fetch")
	var concurrency = flag.Int("concurrency", 3, "bounded worker concurrency")
	var pollInterval = flag.Duration("poll-interval", 0, "optional fixed interval between poll rounds (0 = run once)")
	flag.Parse()
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	logger.Printf("data directory %s", *storeRoot)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	client := pricewatch.NewClient(pricewatch.ClientConfig{})
	run := func(round int) error {
		_, err := pricewatch.RunFetch(ctx, pricewatch.RunnerConfig{WatchlistPath: *watchlistPath, StoreRoot: *storeRoot, DryRun: *dryRun, Concurrency: *concurrency, LookbackDays: *lookback, Logger: logger, Client: client})
		if err != nil {
			return err
		}
		if round > 0 {
			logger.Printf("pricewatch poll round=%d complete", round)
		}
		return nil
	}
	if *pollInterval <= 0 {
		if err := run(0); err != nil {
			logger.Fatalf("pricewatch run failed: %v", err)
		}
		return
	}
	for round := 1; ; round++ {
		if err := run(round); err != nil {
			logger.Printf("pricewatch run failed round=%d: %v", round, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(*pollInterval):
		}
	}
}
