package tcpstreamsdk

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

type Client struct {
	cfg Config

	mu      sync.RWMutex
	conn    net.Conn
	closed  bool
	cursor  ReplayCursor
	filters map[string]struct{}
	subs    map[chan Event]struct{}
	rawSubs map[chan RawEvent]struct{}

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(cfg Config) *Client {
	if cfg.InitialReconnectDelay <= 0 {
		cfg.InitialReconnectDelay = 200 * time.Millisecond
	}
	if cfg.MaxReconnectDelay <= 0 {
		cfg.MaxReconnectDelay = 5 * time.Second
	}
	if cfg.HeartbeatTimeout <= 0 {
		cfg.HeartbeatTimeout = 30 * time.Second
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 128
	}
	f := make(map[string]struct{}, len(cfg.Filters))
	for _, v := range cfg.Filters {
		f[v] = struct{}{}
	}
	return &Client{cfg: cfg, subs: make(map[chan Event]struct{}), rawSubs: make(map[chan RawEvent]struct{}), filters: f}
}

func (c *Client) Dial(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	if err := c.connect(); err != nil {
		return err
	}
	c.wg.Add(1)
	go c.loop(ctx)
	return nil
}

func (c *Client) Resume(cursor ReplayCursor) { c.mu.Lock(); c.cursor = cursor; c.mu.Unlock() }

func (c *Client) Subscribe(ctx context.Context) <-chan Event {
	ch := make(chan Event, c.cfg.BufferSize)
	c.mu.Lock()
	c.subs[ch] = struct{}{}
	c.mu.Unlock()
	go func() { <-ctx.Done(); c.mu.Lock(); delete(c.subs, ch); close(ch); c.mu.Unlock() }()
	return ch
}

func (c *Client) SubscribeRaw(ctx context.Context) <-chan RawEvent {
	ch := make(chan RawEvent, c.cfg.BufferSize)
	c.mu.Lock()
	c.rawSubs[ch] = struct{}{}
	c.mu.Unlock()
	go func() { <-ctx.Done(); c.mu.Lock(); delete(c.rawSubs, ch); close(ch); c.mu.Unlock() }()
	return ch
}

func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.mu.Unlock()
	c.wg.Wait()
	return nil
}

func (c *Client) connect() error {
	c.mu.RLock()
	addr := c.cfg.Address
	cursor := c.cursor
	c.mu.RUnlock()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	if cursor.LastEventID != "" {
		_, _ = fmt.Fprintf(conn, "{\"resume_from\":%q}\n", cursor.LastEventID)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	return nil
}

func (c *Client) loop(ctx context.Context) {
	defer c.wg.Done()
	for {
		if err := c.readLoop(ctx); err != nil {
			if errors.Is(err, context.Canceled) || c.isClosed() {
				return
			}
			if c.cfg.OnDisconnect != nil {
				c.cfg.OnDisconnect(err)
			}
			if c.cfg.Metrics != nil {
				c.cfg.Metrics.IncHeartbeatFailures()
			}
			if !c.cfg.Reconnect {
				return
			}
			if recErr := c.reconnect(ctx); recErr != nil {
				return
			}
			continue
		}
		return
	}
}

func (c *Client) readLoop(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	timeout := c.cfg.HeartbeatTimeout
	c.mu.RUnlock()
	if conn == nil {
		return errors.New("no connection")
	}
	s := bufio.NewScanner(conn)
	buf := make([]byte, 0, 1024*64)
	s.Buffer(buf, 1024*1024)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		if !s.Scan() {
			if err := s.Err(); err != nil {
				return err
			}
			return ioEOF{}
		}
		line := append([]byte(nil), s.Bytes()...)
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		if string(line) == "ping" || string(line) == "pong" {
			continue
		}
		c.broadcastRaw(RawEvent{Bytes: line})
		var evt Event
		if err := json.Unmarshal(line, &evt); err != nil {
			if c.cfg.Metrics != nil {
				c.cfg.Metrics.IncDecodeFailures()
			}
			continue
		}
		if evt.ID != "" {
			c.Resume(ReplayCursor{LastEventID: evt.ID})
		}
		if !c.matchFilter(evt.Type) {
			continue
		}
		if !evt.IngestedAt.IsZero() && c.cfg.Metrics != nil {
			c.cfg.Metrics.ObserveLag(time.Since(evt.IngestedAt))
		}
		c.broadcast(evt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

type ioEOF struct{}

func (ioEOF) Error() string { return "eof" }

func (c *Client) reconnect(ctx context.Context) error {
	delay := c.cfg.InitialReconnectDelay
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := c.connect(); err == nil {
			if c.cfg.OnReconnect != nil {
				c.cfg.OnReconnect()
			}
			if c.cfg.Metrics != nil {
				c.cfg.Metrics.IncReconnects()
			}
			return nil
		}
		j := time.Duration(rand.Int63n(int64(delay/2 + 1)))
		t := delay + j
		if t > c.cfg.MaxReconnectDelay {
			t = c.cfg.MaxReconnectDelay
		}
		tm := time.NewTimer(t)
		select {
		case <-ctx.Done():
			tm.Stop()
			return ctx.Err()
		case <-tm.C:
		}
		delay *= 2
		if delay > c.cfg.MaxReconnectDelay {
			delay = c.cfg.MaxReconnectDelay
		}
	}
}

func (c *Client) matchFilter(tp string) bool {
	if len(c.filters) == 0 {
		return true
	}
	_, ok := c.filters[tp]
	return ok
}

func (c *Client) broadcast(evt Event) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for ch := range c.subs {
		if c.cfg.DropPolicy == Block {
			ch <- evt
			continue
		}
		select {
		case ch <- evt:
		default:
			<-ch
			ch <- evt
			if c.cfg.Metrics != nil {
				c.cfg.Metrics.IncDroppedEvents()
			}
		}
	}
}
func (c *Client) broadcastRaw(evt RawEvent) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for ch := range c.rawSubs {
		select {
		case ch <- evt:
		default:
		}
	}
}
func (c *Client) isClosed() bool { c.mu.RLock(); defer c.mu.RUnlock(); return c.closed }

func Decode[T any](evt Event) (T, error) {
	var t T
	err := json.Unmarshal(evt.Data, &t)
	return t, err
}
