package client

import (
	"context"
	"fmt"
	"github.com/Wa4h1h/go-tftp/pkg/types"
	"go.uber.org/zap"
	"net"
	"time"
)

type Connector interface {
	Connect(addr string) error
	Get(ctx context.Context, filename string) error
	Put(ctx context.Context, filename string) error
}

type Client struct {
	conn    net.Conn
	l       *zap.SugaredLogger
	timeout time.Duration
}

func NewClient(l *zap.SugaredLogger) Connector {
	c := &Client{l: l}
	c.timeout = time.Duration(types.DefaultClientTimeout) * time.Second

	return c
}

func (c *Client) SetTimeout(timeout uint) {
	c.timeout = time.Duration(timeout) * time.Second
}

func (c *Client) Connect(addr string) error {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return fmt.Errorf("error while dialing %s: %w", addr, err)
	}

	c.conn = conn

	return nil
}

func (c *Client) Get(ctx context.Context, filename string) error {
	return nil
}

func (c *Client) Put(ctx context.Context, filename string) error {
	return nil
}
