package pricewatch

import (
	"testing"
	"time"
)

func TestParseChartResponse(t *testing.T) {
	body := []byte(`{"chart":{"result":[{"timestamp":[1716768000,1716854400],"indicators":{"quote":[{"open":[100.0,101.0],"high":[110.0,111.0],"low":[90.0,91.0],"close":[105.0,null],"volume":[123,456]}],"adjclose":[{"adjclose":[104.5,100.0]}]}}]}}`)
	candles, err := parseChartResponse("AAPL", body)
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 {
		t.Fatalf("expected 1 candle got %d", len(candles))
	}
	c := candles[0]
	if c.Ticker != "AAPL" || c.Date.Format("2006-01-02") != "2024-05-27" {
		t.Fatalf("unexpected candle identity: %+v", c)
	}
	if c.Close != 105.0 || c.AdjClose != 104.5 || c.Volume != 123 {
		t.Fatalf("unexpected values: %+v", c)
	}
	if c.Date.Hour() != 0 || c.Date.Location() != time.UTC {
		t.Fatalf("expected utc midnight got %v", c.Date)
	}
}
