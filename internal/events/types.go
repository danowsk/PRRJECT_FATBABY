package events

// EventTypePriceCandleDaily is the Type string for daily OHLCV candle events.
const EventTypePriceCandleDaily = "price_candle_daily"

type PriceCandleDaily struct {
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	AdjClose float64 `json:"adj_close"`
	Volume   int64   `json:"volume"`
}

// EventTypeFundamentalSnapshot is the Type string for XBRL fundamental snapshots.
const EventTypeFundamentalSnapshot = "fundamental_snapshot"

type FundamentalSnapshot struct {
	Revenue         float64 `json:"revenue,omitempty"`
	NetIncome       float64 `json:"net_income,omitempty"`
	EPS             float64 `json:"eps,omitempty"`
	GrossMargin     float64 `json:"gross_margin,omitempty"`
	OperatingIncome float64 `json:"operating_income,omitempty"`
	Period          string  `json:"period,omitempty"`            // e.g. "2026Q1"
	FiscalPeriodEnd string  `json:"fiscal_period_end,omitempty"` // RFC3339
}

// EventTypeSignalEmitted is the Type string for processor-generated signals.
const EventTypeSignalEmitted = "signal_emitted"
