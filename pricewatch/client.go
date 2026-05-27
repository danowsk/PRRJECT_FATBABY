package pricewatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ClientConfig struct {
	Timeout    time.Duration
	MaxRetries int
	UserAgent  string
}

type Client struct {
	http               *http.Client
	maxRetries         int
	userAgent, baseURL string
}

func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "prrject-fatbaby-pricewatch/0.1"
	}
	return &Client{http: &http.Client{Timeout: cfg.Timeout}, maxRetries: cfg.MaxRetries, userAgent: cfg.UserAgent, baseURL: "https://query1.finance.yahoo.com"}
}

func (c *Client) FetchDailyCandles(ctx context.Context, ticker string, lookbackDays int) ([]DailyCandle, error) {
	if lookbackDays <= 0 {
		lookbackDays = 7
	}
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	u := fmt.Sprintf("%s/v8/finance/chart/%s?interval=1d&range=%dd&includeAdjustedClose=true", c.baseURL, url.PathEscape(ticker), lookbackDays)
	body, err := c.do(ctx, u)
	if err != nil {
		return nil, err
	}
	return parseChartResponse(ticker, body)
}

func (c *Client) do(ctx context.Context, u string) ([]byte, error) {
	if c == nil || c.http == nil {
		return nil, errors.New("nil client")
	}
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("User-Agent", c.userAgent)
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
		} else {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return b, nil
			}
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("http %d", resp.StatusCode)
			} else {
				return nil, fmt.Errorf("request failed status=%d body=%s", resp.StatusCode, string(b))
			}
		}
		if attempt == c.maxRetries {
			break
		}
		sleep := time.Duration(math.Min(float64(500*time.Millisecond)*math.Pow(2, float64(attempt)), float64(8*time.Second)))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(sleep):
		}
	}
	return nil, fmt.Errorf("request failed after retries: %w", lastErr)
}

type chartResp struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open, High, Low, Close []*float64
					Volume                 []int64
				} `json:"quote"`
				AdjClose []struct {
					AdjClose []*float64 `json:"adjclose"`
				} `json:"adjclose"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

func parseChartResponse(ticker string, body []byte) ([]DailyCandle, error) {
	var resp chartResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode yahoo response: %w", err)
	}
	if len(resp.Chart.Result) == 0 || len(resp.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, nil
	}
	r := resp.Chart.Result[0]
	q := r.Indicators.Quote[0]
	var adj []*float64
	if len(r.Indicators.AdjClose) > 0 {
		adj = r.Indicators.AdjClose[0].AdjClose
	}
	out := make([]DailyCandle, 0, len(r.Timestamp))
	for i, ts := range r.Timestamp {
		if i >= len(q.Close) || q.Close[i] == nil || *q.Close[i] == 0 {
			continue
		}
		if i >= len(q.Open) || i >= len(q.High) || i >= len(q.Low) || q.Open[i] == nil || q.High[i] == nil || q.Low[i] == nil {
			continue
		}
		c := DailyCandle{Ticker: ticker, Date: dayUTC(time.Unix(ts, 0).UTC()), Open: *q.Open[i], High: *q.High[i], Low: *q.Low[i], Close: *q.Close[i]}
		if i < len(q.Volume) {
			c.Volume = q.Volume[i]
		}
		if i < len(adj) && adj[i] != nil {
			c.AdjClose = *adj[i]
		} else {
			c.AdjClose = c.Close
		}
		out = append(out, c)
	}
	return out, nil
}

func dayUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
