package eventsink

import "errors"

var (
	ErrNilPrimarySink    = errors.New("primary sink is required")
	ErrSecondaryQueueFull = errors.New("secondary queue is full")
)
