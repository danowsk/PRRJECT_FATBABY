package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/example/prrject-fatbaby/pkg/fatstream"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9001", "tcp address")
	types := flag.String("type", "", "comma-separated event types")
	raw := flag.Bool("raw", false, "print raw json")
	resume := flag.String("resume", "", "resume from event id")
	flag.Parse()

	cfg := fatstream.Config{Address: *addr, Reconnect: true}
	if *types != "" {
		cfg.Filters = strings.Split(*types, ",")
	}
	c, err := fatstream.Dial(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	if *resume != "" {
		c.Resume(fatstream.ReplayCursor{LastEventID: *resume})
	}

	ctx := context.Background()
	if *raw {
		for evt := range c.SubscribeRaw(ctx) {
			fmt.Println(string(evt.Bytes))
		}
		return
	}
	for evt := range c.Subscribe(ctx) {
		b, _ := json.MarshalIndent(evt, "", "  ")
		fmt.Println(string(b))
	}
}
