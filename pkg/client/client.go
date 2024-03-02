package client

import (
	"context"
	"errors"
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
	SetTrace()
	SetTimeout(timeout uint)
	execute(filename string, op Op) error
	Get(filename string) error
	Put(filename string) error
}

type Client struct {
	remoteAddr *net.UDPAddr
	l          *zap.SugaredLogger
	timeout    time.Duration
	numTries   uint
	trace      bool
}

func NewClient(l *zap.SugaredLogger, numTries uint) Connector {
	c := &Client{l: l, numTries: numTries}
	c.timeout = time.Duration(types.DefaultClientTimeout) * time.Second

	return c
}

func (c *Client) SetTrace() {
	c.trace = !c.trace
}

func (c *Client) SetTimeout(timeout uint) {
	c.timeout = time.Duration(timeout) * time.Second
}

func (c *Client) execute(filename string, op Op) error {
	var err error

	done := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

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

		b, errM := req.MarshalBinary()
		if errM != nil {
			d <- fmt.Errorf("error while marshalling request: %w", errM)

			return
		}

		if _, err := conn.Write(b); err != nil {
			d <- fmt.Errorf("error while writing request: %w", err)

			return
		}

		t := server.NewTransfer(conn, c.l, c.timeout, c.timeout, int(c.numTries), c.trace)

		switch op {
		case get:
			if err := t.Receive(file); err != nil {
				d <- fmt.Errorf("error while receiving file %s: %w", file, err)
			}
		case put:
			{
				if !checkFileExist(file) {
					d <- fmt.Errorf("%s does not exist", file)

					return
				}

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
						fmt.Println(errPacket.ErrMsg)

						return
					}
				}

				if err = t.Send(file); err != nil {
					d <- fmt.Errorf("error while sending file %s: %w", file, err)
				}
			}
		}

		close(d)
	}(done, filename)

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = errors.New(fmt.Sprintf("request exceeded timeout %ds", int(c.timeout.Seconds())))
		}
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

func (c *Client) Get(filename string) error {
	return c.execute(filename, get)
}

func (c *Client) Put(filename string) error {
	return c.execute(filename, put)
}
