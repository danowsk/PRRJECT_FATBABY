package tcpstreamsdk

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Source       string            `json:"source"`
	OccurredAt   time.Time         `json:"occurred_at"`
	IngestedAt   time.Time         `json:"ingested_at"`
	PartitionKey string            `json:"partition_key,omitempty"`
	Data         json.RawMessage   `json:"data"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type RawEvent struct{ Bytes []byte }

type ReplayCursor struct{ LastEventID string }

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type Metrics interface {
	IncReconnects()
	IncDroppedEvents()
	IncHeartbeatFailures()
	IncDecodeFailures()
	ObserveLag(time.Duration)
}

type DropPolicy int

const (
	DropOldest DropPolicy = iota
	Block
)

type Config struct {
	Address               string
	Reconnect             bool
	MaxReconnectDelay     time.Duration
	InitialReconnectDelay time.Duration
	HeartbeatTimeout      time.Duration
	BufferSize            int
	Logger                Logger
	OnReconnect           func()
	OnDisconnect          func(error)
	Filters               []string
	Metrics               Metrics
	DropPolicy            DropPolicy
}
