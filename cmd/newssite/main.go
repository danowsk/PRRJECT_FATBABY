package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/prrject-fatbaby/eventstore"
	"github.com/example/prrject-fatbaby/internal/newssite"
)

func main() {
	storeRoot := flag.String("store", "var/secwatch", "path to eventstore root")
	addr := flag.String("addr", ":8082", "listen address")
	readTimeout := flag.Duration("read-timeout", 10*time.Second, "")
	writeTimeout := flag.Duration("write-timeout", 15*time.Second, "")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	store, err := eventstore.NewFileStore(*storeRoot)
	if err != nil {
		logger.Fatalf("open store: %v", err)
	}
	defer store.Close()

	h := newssite.NewHandler(store, logger)
	srv := &http.Server{
		Addr:         *addr,
		Handler:      h,
		ReadTimeout:  *readTimeout,
		WriteTimeout: *writeTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	logger.Printf("newssite listening addr=%s store=%s", *addr, *storeRoot)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("listen: %v", err)
	}
}
