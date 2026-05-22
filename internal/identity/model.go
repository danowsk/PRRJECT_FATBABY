package identity

type SecurityRef struct {
	Exchange    string  `json:"exchange,omitempty"`
	Symbol      string  `json:"symbol,omitempty"`
	CIK         string  `json:"cik,omitempty"`
	Source      string  `json:"source"`
	Confidence  float32 `json:"confidence"`
	MatchedText string  `json:"matched_text,omitempty"`
}

type DiscoveryIdentity struct {
	PrimaryTicker *SecurityRef  `json:"primary_ticker,omitempty"`
	AllTickers    []SecurityRef `json:"all_tickers,omitempty"`
}
