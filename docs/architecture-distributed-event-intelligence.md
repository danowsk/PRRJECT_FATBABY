# Distributed Event Intelligence Architecture

This document captures the next-stage architecture evolution for `prrject-fatbaby`: from a single-node scraper pipeline into a distributed event intelligence platform.

## Core principle

Do **not** discard the local append-only event store.

The local append-only event log remains the **source of truth per node** for ingest durability and deterministic debugging.

S3 should be introduced as a **durable distributed archive + replay substrate**, not as the initial primary ingest write path.

Preferred ingest durability flow:

```text
ingest
  ↓
local durable append
  ↓
async replication
  ↓
S3
```

Avoid this at first:

```text
ingest
  ↓
S3 directly
```

Direct-to-S3 primaries couple ingestion reliability to cloud-network behavior (latency, retries, rate limits, transient auth failures, and partitions).

## Architecture shift

Current shape:

```text
poller
  ↓
append ndjson
  ↓
processor reads ndjson
```

Target shape:

```text
discoverers
    ↓
event bus
    ↓
fanout router
    ├── local append store
    ├── S3 object archive
    ├── realtime processors
    ├── embedding workers
    ├── alerting
    ├── graph builders
    ├── ML/ranking
    ├── replay streams
    └── analytics lake
```

This is a distributed event system. Treat immutable events as the contract between independently scalable components.

## Recommended phased evolution

### Phase 1 (current): single node + local NDJSON

Keep this model. It is ideal for early deterministic operation, replayability, and incident debugging.

### Phase 2: dual-sink fanout

Add a sink abstraction:

```go
type EventSink interface {
    Write(ctx context.Context, evt Event) error
}
```

Initial implementations:

- `FileSink`
- `S3Sink`
- `KafkaSink` (future)
- `NATSSink` (future)
- `RedpandaSink` (future)

Fanout writer shape:

```go
type FanoutSink struct {
    Primary   EventSink
    Secondary []EventSink
}
```

Behavioral rules:

1. **Primary write must succeed** (initially `FileSink`).
2. Secondary writes are async/retriable.
3. Secondary failures route to dead-letter handling.
4. Use bounded concurrency + retry policy (for example via `errgroup` with semaphores).

Suggested package scaffold:

```text
internal/eventsink/
    sink.go
    fanout.go
    file_sink.go
    s3_sink.go
```

### Phase 3: deterministic S3 object layout

Use partitioned object paths; do not write arbitrary flat filenames.

Example:

```text
s3://fatbaby-events/
    source=sec/
        year=2026/
            month=05/
                day=21/
                    hour=17/
                        event-uuid.json.zst

    source=prnewswire/
        year=2026/
            month=05/
                day=21/
```

Benefits:

- Athena-friendly partitioning
- Replay-friendly prefix scanning
- Better lifecycle + retention control
- Easier conversion to Parquet/Lakehouse patterns

### Phase 4: canonical event envelope

Standardize a universal envelope used across all producers/consumers:

```go
type Event struct {
    ID           string            `json:"id"`
    Type         string            `json:"type"`

    Source       string            `json:"source"`

    OccurredAt   time.Time         `json:"occurred_at"`
    IngestedAt   time.Time         `json:"ingested_at"`

    PartitionKey string            `json:"partition_key,omitempty"`

    Identity     DiscoveryIdentity `json:"identity,omitempty"`

    Data         json.RawMessage   `json:"data"`

    Metadata     map[string]string `json:"metadata,omitempty"`
}
```

This envelope is the cross-service contract for replay, enrichment, analytics, and training.

### Phase 5: Kubernetes transition

Kubernetes becomes the control plane once workload fanout includes multiple autonomous services:

- SEC pollers
- PR/RSS pollers
- HTML extraction workers
- ticker/entity resolvers
- embedding workers
- summarizers/rerankers
- alert engines
- replay processors
- vector DB updaters
- archive exporters

Design principle: keep most processors stateless.

**Stateful services**:

- eventstore writers
- object archival
- queue backends
- vector DB
- Postgres
- Redis
- Kafka/Redpanda/NATS persistence

**Stateless services**:

- parsers
- enrichers
- NLP/extraction
- embeddings
- summarization/analyzers

## Bus strategy guidance

At scale, file-tail-only consumers become operationally brittle. Introduce a real bus for decoupled subscription and backpressure isolation.

Recommended progression:

1. Start with **NATS JetStream** (operational simplicity).
2. Move to **Redpanda/Kafka** only if scale/retention/compliance requirements outgrow NATS.

## Why event-first decoupling matters

Avoid tightly coupled processor RPC webs.

Prefer communication via:

- immutable events
- queues/streams
- object storage
- append-only logs

This preserves replayability, failure isolation, and independent scaling.

## Emily agent operational role (future)

As the platform grows, orchestration agents can automate operations workflows:

- monitor ingestion lag
- inspect consumer health
- trigger dead-letter replay
- scale enrichment workloads
- detect stream drift/anomalies
- run temporal diagnostics

This enables an autonomous intelligence operations layer on top of the event platform.

## High-value capability unlocked by S3 replication

Once async S3 archival is in place, you gain cluster-wide historical replay:

```text
new processor deployed
    ↓
replay historical events from S3
    ↓
retroactively compute new intelligence layer
```

That is a strategic multiplier: every future algorithm can learn from historical ingestion without re-crawling source systems.
