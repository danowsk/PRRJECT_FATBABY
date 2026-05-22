package fatstream

import (
	"context"

	"github.com/example/prrject-fatbaby/internal/tcpstreamsdk"
)

type Client struct{ inner *tcpstreamsdk.Client }

func Dial(cfg Config) (*Client, error) {
	c := tcpstreamsdk.New(cfg)
	if err := c.Dial(context.Background()); err != nil {
		return nil, err
	}
	return &Client{inner: c}, nil
}
func (c *Client) Subscribe(ctx context.Context) <-chan Event       { return c.inner.Subscribe(ctx) }
func (c *Client) SubscribeRaw(ctx context.Context) <-chan RawEvent { return c.inner.SubscribeRaw(ctx) }
func (c *Client) Resume(cursor ReplayCursor)                       { c.inner.Resume(cursor) }
func (c *Client) Close() error                                     { return c.inner.Close() }
