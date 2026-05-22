package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/prrject-fatbaby/broker"
	"github.com/example/prrject-fatbaby/eventstore"
	"github.com/example/prrject-fatbaby/feedserver"
)

func main() {
	var addr, storePath, routes, cert, key string
	var maxConns, maxBuffer int
	var heartbeat, poll, writeTimeout time.Duration
	flag.StringVar(&addr, "addr", ":7070", "")
	flag.StringVar(&storePath, "store", "var/feedserver", "")
	flag.StringVar(&routes, "routes", "config/routes.json", "")
	flag.StringVar(&cert, "tls-cert", "", "")
	flag.StringVar(&key, "tls-key", "", "")
	flag.IntVar(&maxConns, "max-conns", 1024, "")
	flag.DurationVar(&heartbeat, "heartbeat", 15*time.Second, "")
	flag.DurationVar(&poll, "poll-interval", 500*time.Millisecond, "")
	flag.IntVar(&maxBuffer, "max-buffer", 512, "")
	flag.DurationVar(&writeTimeout, "write-timeout", 5*time.Second, "")
	flag.Parse()
	store, err := eventstore.NewFileStore(storePath)
	if err != nil {
		log.Fatal(err)
	}
	reg, err := broker.NewRegistry(routes)
	if err != nil {
		log.Fatal(err)
	}
	var tlsCfg *tls.Config
	if cert != "" && key != "" {
		c, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			log.Fatal(err)
		}
		tlsCfg = &tls.Config{Certificates: []tls.Certificate{c}}
	}
	srv := feedserver.NewServer(feedserver.ServerConfig{Addr: addr, Store: store, Registry: reg, MaxConns: maxConns, TLSConfig: tlsCfg, SessionConfig: feedserver.SessionConfig{HeartbeatInterval: heartbeat, PollInterval: poll, MaxBufferDepth: maxBuffer, WriteTimeout: writeTimeout}, Logger: log.Default()})
	log.Printf("feedserver startup addr=%s store=%s routes=%s maxConns=%d", addr, storePath, routes, maxConns)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGHUP)
		for range ch {
			_ = reg.Reload()
		}
	}()
	if err := srv.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
	}
}
