package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/example/prrject-fatbaby/pkg/fatstream"
)

type metrics struct{}

func (metrics) IncReconnects()           {}
func (metrics) IncDroppedEvents()        {}
func (metrics) IncHeartbeatFailures()    {}
func (metrics) IncDecodeFailures()       {}
func (metrics) ObserveLag(time.Duration) {}

func main() {
	client, err := fatstream.Dial(fatstream.Config{Address: "127.0.0.1:9001", Reconnect: true, Metrics: metrics{}})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	for evt := range client.Subscribe(context.Background()) {
		fmt.Println(evt.Type)
	}
}
