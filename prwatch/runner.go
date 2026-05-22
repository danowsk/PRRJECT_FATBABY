package prwatch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
	identitypkg "github.com/example/prrject-fatbaby/internal/identity"
	prid "github.com/example/prrject-fatbaby/internal/prwatch"
)

type Logger interface {
	Printf(format string, args ...any)
}

type RunnerConfig struct {
	StoreRoot string
	DryRun    bool
	Now       func() time.Time
	Logger    Logger
	Client    *Client
}

type Summary struct {
	SeenSkipped int
	Discovered  int
}

type PressReleaseDiscovered struct {
	URL              string                        `json:"url"`
	Source           string                        `json:"source"`
	DiscoveredAt     time.Time                     `json:"discovered_at"`
	Identity         identitypkg.DiscoveryIdentity `json:"identity"`
	ExtractionMethod string                        `json:"extraction_method"`
	RawBodySnippet   string                        `json:"raw_body_snippet,omitempty"`
	ContentHash      string                        `json:"content_hash"`
	Metadata         map[string]string             `json:"metadata,omitempty"`
	Headline         string                        `json:"headline,omitempty"`
	Company          string                        `json:"company,omitempty"`
	PublishedAt      string                        `json:"published_at,omitempty"`
}

func RunDiscovery(ctx context.Context, cfg RunnerConfig) (Summary, error) {
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.Client == nil {
		cfg.Client = NewClient(ClientConfig{})
	}
	disc, err := cfg.Client.Discover(ctx)
	if err != nil {
		return Summary{}, err
	}
	store, err := eventstore.NewFileStore(cfg.StoreRoot)
	if err != nil {
		return Summary{}, fmt.Errorf("open event store: %w", err)
	}
	defer store.Close()

	seen, err := LoadSeenIDs(ctx, store)
	if err != nil {
		return Summary{}, err
	}
	s := Summary{}
	for _, pr := range disc {
		if _, ok := seen[pr.ID]; ok {
			s.SeenSkipped++
			continue
		}
		s.Discovered++
		if cfg.DryRun {
			continue
		}
		ev := eventstore.Event{
			ID:           "pr_discovered:" + pr.ID,
			Type:         "pr_discovered",
			OccurredAt:   cfg.Now(),
			AggregateKey: pr.ID,
			Source:       "prnewswire",
			Data:         mustJSON(eventData(ctx, cfg, pr, cfg.Now())),
		}
		if _, err := store.Append(ctx, ev); err != nil {
			return s, fmt.Errorf("append event %s: %w", pr.ID, err)
		}
		seen[pr.ID] = struct{}{}
	}
	if cfg.Logger != nil {
		cfg.Logger.Printf("prwatch summary discovered=%d seen=%d dry_run=%t", s.Discovered, s.SeenSkipped, cfg.DryRun)
	}
	return s, nil
}

func eventData(ctx context.Context, cfg RunnerConfig, pr PRDiscovery, now time.Time) PressReleaseDiscovered {
	e := PressReleaseDiscovered{URL: pr.URL, Source: "prnewswire", DiscoveredAt: now.UTC(), ExtractionMethod: "regex", Headline: pr.Headline, Company: pr.Company}
	if !pr.Timestamp.IsZero() {
		e.PublishedAt = pr.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	if refs, snippet := discoverTickers(ctx, cfg.Client, pr.URL); len(refs) > 0 {
		e.Identity.AllTickers = refs
		first := refs[0]
		e.Identity.PrimaryTicker = &first
		e.RawBodySnippet = snippet
	}
	e.Metadata = map[string]string{"id": pr.ID}
	b, _ := json.Marshal(e)
	sum := sha256.Sum256(b)
	e.ContentHash = hex.EncodeToString(sum[:])
	return e
}

func LoadSeenIDs(ctx context.Context, store eventstore.EventStore) (map[string]struct{}, error) {
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
			if rec.Event.Type != "pr_discovered" {
				continue
			}
			if rec.Event.AggregateKey != "" {
				seen[rec.Event.AggregateKey] = struct{}{}
				continue
			}
			var e PressReleaseDiscovered
			if err := json.Unmarshal(rec.Event.Data, &e); err == nil && e.URL != "" {
				seen[e.URL] = struct{}{}
			}
		}
		from = recs[len(recs)-1].Sequence + 1
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func discoverTickers(ctx context.Context, c *Client, u string) ([]identitypkg.SecurityRef, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, ""
	}
	req.Header.Set("User-Agent", c.ua)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	refs := prid.ExtractFromHTML(body)
	snippet := string(body)
	if len(snippet) > 256 {
		snippet = snippet[:256]
	}
	return refs, snippet
}
