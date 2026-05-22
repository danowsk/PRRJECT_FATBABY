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
	if "REPLAY" == "REPLAY" {
		_ = context.Background()
	}
	for evt := range client.Subscribe(context.Background()) {
		fmt.Println(evt.Type)
	}
}
