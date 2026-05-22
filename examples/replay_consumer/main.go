package main

import (
	"context"
	"fmt"
	"log"

	"github.com/example/prrject-fatbaby/pkg/fatstream"
)

func main() {
	client, err := fatstream.Dial(fatstream.Config{Address: "127.0.0.1:9001", Reconnect: true})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	client.Resume(fatstream.ReplayCursor{LastEventID: "evt-1000"})
	for evt := range client.Subscribe(context.Background()) {
		fmt.Println(evt.ID, evt.Type)
	}
}
