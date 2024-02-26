package client

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Wa4h1h/go-tftp/pkg/server"
	"github.com/Wa4h1h/go-tftp/pkg/types"
	"go.uber.org/zap"
)

type Connector interface {
	Connect(addr string) error
	Close() error
	Get(ctx context.Context, filename string) error
	Put(ctx context.Context, filename string) error
}

type Client struct {
	conn     net.Conn
	l        *zap.SugaredLogger
	timeout  time.Duration
	numTries uint
}

func NewClient(l *zap.SugaredLogger, numTries uint) Connector {
	c := &Client{l: l, numTries: numTries}
	c.timeout = time.Duration(types.DefaultClientTimeout) * time.Second

	return c
}

func (c *Client) SetTimeout(timeout uint) {
	c.timeout = time.Duration(timeout) * time.Second
}

func (c *Client) Connect(addr string) error {
	conn, errListen := net.Dial("udp", addr)
	if errListen != nil {
		return fmt.Errorf("error while listening %s: %w", addr, errListen)
	}

	c.conn = conn

	return nil
}

func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("error while closing socket: %w", err)
	}

	return nil
}

func (c *Client) Get(ctx context.Context, filename string) error {
	var cancel context.CancelFunc
	var err error

	done := make(chan error)

	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	go func(d chan<- error, file string) {
		req := &types.Request{
			Opcode:   types.OpCodeRRQ,
			Filename: file,
			Mode:     types.DefaultMode,
		}

		b, err := req.MarshalBinary()
		if err != nil {
			d <- fmt.Errorf("error while marshalling request: %w", err)

			return
		}

		if _, err := c.conn.Write(b); err != nil {
			d <- fmt.Errorf("error while writing request: %w", err)

			return
		}

		t := server.NewTransfer(c.conn, c.l, c.timeout, c.timeout, int(c.numTries))

		if err := t.Receive(file); err != nil {
			d <- fmt.Errorf("error while receiving file %s: %w", file, err)

			return
		}

		close(d)
	}(done, filename)

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-done:
	}

	return err
}

func (c *Client) Put(ctx context.Context, filename string) error {
	return nil
}
