package feedserver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"sync"
	"sync/atomic"

	"github.com/example/prrject-fatbaby/broker"
	"github.com/example/prrject-fatbaby/eventstore"
)

type ServerConfig struct {
	Addr          string
	Store         eventstore.EventStore
	Registry      *broker.Registry
	Logger        Logger
	MaxConns      int
	TLSConfig     *tls.Config
	SessionConfig SessionConfig
}
type Stats struct{ TotalConnections, TotalFramesSent, TotalBytesOut int64 }
type Server struct {
	cfg       ServerConfig
	sem       chan struct{}
	sessions  sync.Map
	active    atomic.Int64
	totalConn atomic.Int64
}

func NewServer(cfg ServerConfig) *Server {
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 1024
	}
	return &Server{cfg: cfg, sem: make(chan struct{}, cfg.MaxConns)}
}
func (s *Server) ActiveSessions() int { return int(s.active.Load()) }
func (s *Server) Stats() Stats        { return Stats{TotalConnections: s.totalConn.Load()} }
func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	go func() { <-ctx.Done(); _ = ln.Close() }()
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		select {
		case s.sem <- struct{}{}:
		default:
			_ = WriteFrame(c, TypeError, []byte(`{"code":"overloaded"}`))
			_ = c.Close()
			continue
		}
		s.totalConn.Add(1)
		s.active.Add(1)
		go func(conn net.Conn) {
			defer func() { <-s.sem; s.active.Add(-1) }()
			cfg := s.cfg.SessionConfig
			cfg.Conn = conn
			cfg.Store = s.cfg.Store
			cfg.Registry = s.cfg.Registry
			sess := NewSession(cfg)
			_ = sess.Run()
		}(c)
	}
}
func sendErrorConn(c net.Conn, code, msg string) {
	b, _ := json.Marshal(map[string]string{"code": code, "message": msg})
	_ = WriteFrame(c, TypeError, b)
}
