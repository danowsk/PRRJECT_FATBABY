# prrject-fatbaby

`prrject-fatbaby` is a Go-based toolkit for financial signal intelligence and event-driven data processing.

It combines:

- a durable, file-backed append-only event store,
- conservative discovery workers for SEC filings and PR Newswire press releases,
- a processing pipeline that turns source documents into structured financial signals, and
- a lightweight dashboard + SSE server for real-time streaming.

## Core capabilities

- **Event store** (`eventstore/`):
  - Append-only persistence with monotonically increasing global sequence numbers.
  - UTC-date-partitioned journal files.
  - Sequence state file for fast recovery after restart.
  - Read-from-sequence APIs for downstream consumers.
- **SEC discovery** (`secwatch/`, `cmd/secwatch`):
  - Polls SEC EDGAR submissions for issuers/forms defined in `config/watchlist.json`.
  - Emits `filing_discovered` events into the event store.
  - Supports dry-run, bounded concurrency, rate limiting, retries, and optional polling loops.
- **Press release discovery** (`prwatch/`, `cmd/prwatch`):
  - Polls PR Newswire, discovers release URLs, and emits discovery events.
  - Supports dry-run and continuous polling.
- **Signal intelligence worker** (`internal/processor`, `pkg/intelligence`, `cmd/processor`):
  - Watches filing discovery events.
  - Fetches source filing documents (HTML/XBRL-linked pages), cleans text, and runs provider analysis.
  - Emits structured signal outputs for downstream trading/monitoring workflows.
- **Realtime streaming + dashboard** (`internal/server`, `cmd/dashboard`):
  - Exposes an HTTP dashboard and SSE stream for newly appended events/signals.

## Repository components

| Component | Description |
| --- | --- |
| `eventstore` | Core storage engine, including persistence/recovery and sequence-based reads. |
| `secwatch` | SEC submission discovery and normalization logic. |
| `prwatch` | PR discovery and source crawling support. |
| `internal/processor` | Event-to-signal processing worker pipeline. |
| `pkg/intelligence` | Signal schema + provider interface for analysis backends. |
| `secfixtures` | Fixture management helpers for SEC corpus snapshots and regression support. |
| `internal/server` | Dashboard + SSE serving layer. |
| `fixtures` | Local issuer fixture corpus (metadata + filing artifacts). |

## Signal schema

Processor-generated signals follow this structure (`pkg/intelligence`):

- `signal_type`: e.g. `M&A`, `Earnings`, `Legal`, `Leadership`, `Other`
- `importance`: integer score from `1` to `10`
- `sentiment`: decimal score from `-1.0` to `1.0`
- `summary`: concise one-sentence event summary
- `impact_analysis`: short paragraph describing likely financial impact
- `raw_metadata`: provider/debug metadata map

## Getting started

### Requirements

- Go `1.22+`

### Install dependencies

```bash
go mod download
```

### Configure watchlist

Maintain issuers/forms in:

- `config/watchlist.json`

Each entry is intended to track a ticker, CIK, and allowed form set (for example `8-K`, `10-Q`, `10-K`).

## Running the system

### 1) SEC discovery

Run a one-shot dry-run poll (no persistence):

```bash
go run ./cmd/secwatch \
  -watchlist ./config/watchlist.json \
  -store ./var/secwatch \
  -dry-run
```

Run with persistence enabled:

```bash
go run ./cmd/secwatch \
  -watchlist ./config/watchlist.json \
  -store ./var/secwatch
```

Run continuously every 5 minutes:

```bash
go run ./cmd/secwatch \
  -watchlist ./config/watchlist.json \
  -store ./var/secwatch \
  -poll-interval 5m
```

### 2) PR discovery

Run PR Newswire discovery loop:

```bash
go run ./cmd/prwatch \
  -store ./var/prwatch
```

Optional dry-run mode:

```bash
go run ./cmd/prwatch \
  -store ./var/prwatch \
  -dry-run
```

### 3) Signal processor

Start processing discovered filings into structured signals:

```bash
go run ./cmd/processor \
  -store ./var/secwatch \
  -workers 4
```

### 4) Dashboard + SSE server

Start dashboard server:

```bash
go run ./cmd/dashboard \
  -data-dir ./var/secwatch \
  -port 8080
```

Then open `http://localhost:8080`.

## Event store quick start

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
)

func main() {
	store, err := eventstore.NewFileStore("./data")
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	payload, _ := json.Marshal(map[string]any{"order_id": "A123", "amount": 4200})
	records, err := store.Append(context.Background(), eventstore.Event{
		ID:         "evt-1",
		Type:       "order.created",
		OccurredAt: time.Now().UTC(),
		Data:       payload,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("appended sequence: %d\n", records[0].Sequence)
}
```

## Testing

```bash
go test ./...
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).
