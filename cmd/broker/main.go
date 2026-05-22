package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/example/prrject-fatbaby/broker"
	"github.com/example/prrject-fatbaby/eventstore"
)

func main() {
	routes := flag.String("routes", filepath.Join("config", "routes.json"), "path to routes JSON file")
	storeRoot := flag.String("store", filepath.Join("var", "broker"), "eventstore root")
	addr := flag.String("addr", ":9090", "listen address")
	flushBytes := flag.Int("flush-bytes", 4096, "stream flush threshold")
	readTimeout := flag.Duration("read-timeout", 30*time.Second, "read timeout")
	writeTimeout := flag.Duration("write-timeout", 0, "write timeout")
	idleTimeout := flag.Duration("idle-timeout", 90*time.Second, "idle timeout")
	flag.Parse()
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	reg, err := broker.LoadRegistry(*routes)
	if err != nil {
		logger.Fatalf("load routes: %v", err)
	}
	store, err := eventstore.NewFileStore(*storeRoot)
	if err != nil {
		logger.Fatalf("store: %v", err)
	}
	defer store.Close()
	tr := &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 30 * time.Second, MaxIdleConnsPerHost: 128, DisableCompression: true, ForceAttemptHTTP2: true}
	ph := &broker.ProxyHandler{Client: &http.Client{Transport: tr}, Store: store, Logger: logger, FlushBytes: *flushBytes}

	mux := http.NewServeMux()
	mux.Handle("/internal/healthz", broker.HealthHandler(reg))
	mux.Handle("/internal/metricz", ph.MetricsHandler())
	mux.Handle("/", broker.AuthMiddleware(reg, ph))
	srv := &http.Server{Addr: *addr, Handler: mux, ReadTimeout: *readTimeout, WriteTimeout: *writeTimeout, IdleTimeout: *idleTimeout, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGHUP)
		for range ch {
			if err := reg.Reload(); err != nil {
				logger.Printf("reload failed: %v", err)
			} else {
				logger.Printf("registry reloaded")
			}
		}
	}()
	go func() {
		<-signalChan(syscall.SIGTERM, os.Interrupt)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	logger.Printf("broker listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("listen: %v", err)
	}
}

func signalChan(sigs ...os.Signal) <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	return ch
}
