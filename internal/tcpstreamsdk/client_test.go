package tcpstreamsdk

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func startServer(t *testing.T, handler func(net.Conn)) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handler(c)
		}
	}()
	return ln.Addr().String()
}

func TestMalformedJSON(t *testing.T) {
	addr := startServer(t, func(c net.Conn) {
		defer c.Close()
		fmt.Fprintln(c, "not-json")
		fmt.Fprintln(c, `{"id":"1","type":"ok","data":{}}`)
	})
	cl := New(Config{Address: addr, BufferSize: 4})
	if err := cl.Dial(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ch := cl.Subscribe(ctx)
	select {
	case ev := <-ch:
		if ev.ID != "1" {
			t.Fatal(ev)
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func TestConcurrentSubscribers(t *testing.T) {
	addr := startServer(t, func(c net.Conn) {
		defer c.Close()
		for i := 0; i < 3; i++ {
			fmt.Fprintf(c, `{"id":"%d","type":"ok","data":{}}`+"\n", i)
		}
		time.Sleep(50 * time.Millisecond)
	})
	cl := New(Config{Address: addr, BufferSize: 8})
	if err := cl.Dial(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := 0
			for range cl.Subscribe(ctx) {
				got++
				if got == 3 {
					return
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkDecode(b *testing.B) {
	e := Event{Data: []byte(`{"x":1,"y":"a"}`)}
	type p struct {
		X int    `json:"x"`
		Y string `json:"y"`
	}
	for i := 0; i < b.N; i++ {
		_, _ = Decode[p](e)
	}
}
