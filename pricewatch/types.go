package pricewatch

import "time"

type DailyCandle struct {
	Ticker                           string
	Date                             time.Time
	Open, High, Low, Close, AdjClose float64
	Volume                           int64
}

type RunnerConfig struct {
	WatchlistPath, StoreRoot  string
	DryRun                    bool
	Concurrency, LookbackDays int
	Logger                    Logger
	Now                       func() time.Time
	Client                    *Client
}

type Summary struct{ Tickers, Fetched, Persisted, Skipped, Failed int }

type Logger interface {
	Printf(format string, args ...any)
}
