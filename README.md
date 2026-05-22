# prrject-fatbaby

`prrject-fatbaby` is a Go-based toolkit for **financial signal intelligence** and **event-driven market data processing**.  
It combines a lightweight append-only event store with discovery workers for SEC filings and press releases, plus a processing pipeline that generates structured signals from raw documents.

## Core capabilities

- **Durable event store**: file-backed append-only record storage with global monotonically increasing sequence numbers.
- **SEC discovery (`secwatch`)**: conservative polling of SEC EDGAR submissions for configured issuers/forms.
- **Press release discovery (`prwatch`)**: automated PR Newswire discovery and ingestion pipeline.
- **Signal intelligence (`processor`)**: document cleaning + LLM-style structured signal extraction from filings and releases.
- **Realtime streaming/server**: dashboard + Server-Sent Events endpoints for following newly discovered events and generated intelligence.

## Architecture and components

| Component | Description |
| --- | --- |
| `eventstore` | Core storage engine; persists and replays events using sequence state + UTC-date journal files. |
| `secwatch` | Polls SEC submissions and emits `filing_discovered` events to the store. |
| `prwatch` | Discovers press releases and crawls article bodies for downstream processing. |
| `processor` | Converts raw filing/release content into structured financial `Signal` output. |
| `secfixtures` | SEC fixture corpus support for backfill, testing, and parser regression. |
| `server` / dashboard | Web/SSE layer for monitoring the event stream and derived signals. |

## Event store design

The file store is designed to be simple, deterministic, and easy to operate locally:

- Event validation requires `ID`, `Type`, and `Data`.
- Records are append-only and sequence-assigned atomically.
- Journals are partitioned by UTC date for bounded file growth.
- The latest sequence is persisted in a state file for quick recovery.
- Consumers can read forward from any sequence with a configurable limit.

## Getting started

### Requirements

- Go **1.22.0+**

### Install dependencies

```bash
go mod download
```

### Configure monitored issuers

Edit `config/watchlist.json` to manage tracked companies. Entries support ticker, CIK, and allowed SEC forms (for example `8-K`, `10-Q`, `10-K`).

## Running the tools

### SEC discovery (`secwatch`)

Dry-run (safe, no persisted discovered events):

```bash
go run ./cmd/secwatch \
  -watchlist ./config/watchlist.json \
  -store ./var/secwatch \
  -dry-run
```

Persist discovered filings:

```bash
go run ./cmd/secwatch \
  -watchlist ./config/watchlist.json \
  -store ./var/secwatch
```

Continuous polling:

```bash
go run ./cmd/secwatch \
  -watchlist ./config/watchlist.json \
  -store ./var/secwatch \
  -poll-interval 5m
```

### Press release discovery (`prwatch`)

```bash
go run ./cmd/prwatch/main.go --store ./var/prwatch
```

### Signal processing (`processor`)

```bash
go run ./cmd/processor/main.go --store ./var/secwatch --workers 4
```

### Dashboard/server

```bash
go run ./cmd/dashboard/main.go
```

## Signal schema

The processor emits structured financial signals intended for analyst workflows:

- `signal_type`: `M&A`, `Earnings`, `Legal`, `Leadership`, or `Other`
- `importance`: integer 1–10
- `sentiment`: numeric score from `-1.0` to `1.0`
- `summary`: one-sentence event summary
- `impact_analysis`: short explanation of likely business/market impact

## Backfill strategy (Track A)

A practical fixture and discovery backfill approach for watched issuers:

1. Start with shallow breadth (all watched issuers, key forms only).
2. Add selective depth (quarterly chains, annual history).
3. Include amendments (`10-K/A`, `10-Q/A`) for supersession/regression handling.
4. Keep strict per-run caps so ingestion remains deterministic and cost-bounded.

Recommended initial scope:

- Last 2–3 years per issuer
- Forms: `10-K`, `10-Q`, `8-K` (+ amendments)
- Per-run cap: ~25 filings total

## Development

Run all tests:

```bash
go test ./...
```

Project layout pointers:

- `eventstore/types.go`
- `eventstore/file_store.go`
- `secwatch/`
- `prwatch/`
- `processor/`
- `cmd/`

## License

MIT. See [LICENSE](LICENSE).
