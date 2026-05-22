package fatstream

import (
	"github.com/example/prrject-fatbaby/internal/tcpstreamsdk"
)

type Event = tcpstreamsdk.Event
type RawEvent = tcpstreamsdk.RawEvent
type ReplayCursor = tcpstreamsdk.ReplayCursor
type Config = tcpstreamsdk.Config
type Logger = tcpstreamsdk.Logger
type Metrics = tcpstreamsdk.Metrics
type DropPolicy = tcpstreamsdk.DropPolicy

const (
	DropOldest = tcpstreamsdk.DropOldest
	Block      = tcpstreamsdk.Block
)
