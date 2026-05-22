package main

import (
	"context"
	"fmt"
	"log"

	"github.com/example/prrject-fatbaby/pkg/fatstream"
)

func main() {
	client, err := fatstream.Dial(fatstream.Config{Address: "127.0.0.1:9001", Reconnect: true, Filters: []string{"filing_discovered"}})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	for evt := range client.Subscribe(context.Background()) {
		fmt.Println("filtered:", evt.Type)
	}
}
