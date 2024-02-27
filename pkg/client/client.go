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

type Op uint16

const (
	get Op = iota
	put
)

type Connector interface {
	Connect(addr string) error
	execute(ctx context.Context, filename string, op Op) error
	Get(ctx context.Context, filename string) error
	Put(ctx context.Context, filename string) error
}

type Client struct {
	remoteAddr *net.UDPAddr
	l          *zap.SugaredLogger
	timeout    time.Duration
	numTries   uint
}

func NewClient(l *zap.SugaredLogger, numTries uint) Connector {
	c := &Client{l: l, numTries: numTries}
	c.timeout = time.Duration(types.DefaultClientTimeout) * time.Second

	return c
}

func (c *Client) SetTimeout(timeout uint) {
	c.timeout = time.Duration(timeout) * time.Second
}

func (c *Client) execute(ctx context.Context, filename string, op Op) error {
	var cancel context.CancelFunc
	var err error

	done := make(chan error)

	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	go func(d chan<- error, file string) {
		conn, errListen := net.DialUDP("udp", nil, c.remoteAddr)
		if errListen != nil {
			d <- fmt.Errorf("error while creating udp listener: %w", errListen)

			return
		}

		defer func(c *net.UDPConn) {
			conn.Close()
		}(conn)

		req := &types.Request{
			Filename: file,
			Mode:     types.DefaultMode,
		}

		if op == get {
			req.Opcode = types.OpCodeRRQ
		} else {
			req.Opcode = types.OpCodeWRQ
		}

		b, err := req.MarshalBinary()
		if err != nil {
			d <- fmt.Errorf("error while marshalling request: %w", err)

			return
		}

		if _, err := conn.Write(b); err != nil {
			d <- fmt.Errorf("error while writing request: %w", err)

			return
		}

		t := server.NewTransfer(conn, c.l, c.timeout, c.timeout, int(c.numTries))

		switch op {
		case get:
			if err := t.Receive(file); err != nil {
				d <- fmt.Errorf("error while receiving file %s: %w", file, err)

				return
			}
		case put:
			buff := make([]byte, types.DatagramSize)
			if _, err := conn.Read(buff); err != nil {
				d <- fmt.Errorf("error while reading ack: %w", err)

				return
			}

			var ack types.Ack
			var errPacket types.Error

			switch {
			case ack.UnmarshalBinary(buff) == nil:
				{
					if ack.BlockNum != 0 {
						d <- fmt.Errorf("expected BlockNum=0 but got %d", ack.BlockNum)

						return
					}
				}
			case errPacket.UnmarshalBinary(buff) == nil:
				{
					fmt.Println(errPacket)
					close(d)

					return
				}
			}

			if err := t.Send(file); err != nil {
				d <- fmt.Errorf("error while sending file %s: %w", file, err)

				return
			}
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

func (c *Client) Connect(addr string) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("error while listening %s: %w", addr, err)
	}

	c.remoteAddr = remoteAddr

	return nil
}

func (c *Client) Get(ctx context.Context, filename string) error {
	return c.execute(ctx, filename, get)
}

func (c *Client) Put(ctx context.Context, filename string) error {
	return c.execute(ctx, filename, put)
}
