package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/Wa4h1h/go-tftp/pkg/types"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
	"go.uber.org/zap"
)

type Transfer interface {
	Send(file string) error
	SendBlock(block []byte, blockNum uint16) error
	AcknowledgeWrq() error
	Receive(file string) error
	ReceiveBlock(blockW io.Writer) (uint16, uint16, error)
}

type Connection struct {
	conn         net.Conn
	l            *zap.SugaredLogger
	numTries     int
	readTimeout  time.Duration
	writeTimeout time.Duration
	trace        bool
}

func NewTransfer(conn net.Conn,
	logger *zap.SugaredLogger, readTimeout time.Duration,
	writeTimeout time.Duration, numTries int, trace bool,
) Transfer {
	return &Connection{
		conn: conn, l: logger, readTimeout: readTimeout,
		writeTimeout: writeTimeout, numTries: numTries,
		trace: trace,
	}
}

func (c *Connection) AcknowledgeWrq() error {
	ack := &types.Ack{
		Opcode:   types.OpCodeACK,
		BlockNum: 0,
	}

	b, err := ack.MarshalBinary()
	if err != nil {
		c.l.Error(err.Error())

		return utils.ErrPacketMarshall
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		c.l.Errorf("error while setting write timeout: %s", err.Error())

		return utils.ErrCanNotSetWriteTimeout
	}

	if _, err := c.conn.Write(b); err != nil {
		c.l.Errorf("error while writing data packet: %s", err.Error())

		return utils.ErrPacketCanNotBeSent
	}

	return nil
}

func (c *Connection) ReceiveBlock(blockW io.Writer) (uint16, uint16, error) {
	var (
		data          types.Data
		errPacket     types.Error
		wrongBlockNum uint16
		nullBytes     uint16
	)

	datagram := make([]byte, types.DatagramSize)

	for tries := c.numTries; tries > 0; tries-- {
		if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return wrongBlockNum, nullBytes, fmt.Errorf("error while setting read timeout: %w", err)
		}

		n, err := c.conn.Read(datagram)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return wrongBlockNum, nullBytes, nil
			}

			c.l.Errorf("error while reading ack: %s", err.Error())

			continue
		}

		if n < 0 {
			c.l.Debug("0 bytes were read")

			continue
		}

		if err := data.UnmarshalBinary(datagram[:n]); err != nil {
			c.l.Errorf("error while unmarshal data packet: %s", err.Error())

			continue
		} else if errPacket.UnmarshalBinary(datagram[:n]) == nil {
			return wrongBlockNum, nullBytes, utils.ErrPacketCanNotBeSent
		}

		src := bytes.NewBuffer(data.Payload)

		copied, errCopy := io.CopyN(blockW, src, int64(len(data.Payload)))
		if errCopy != nil {
			return wrongBlockNum, nullBytes, fmt.Errorf("error while copy payload: %w", errCopy)
		}

		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return wrongBlockNum, nullBytes, fmt.Errorf("error while setting write timeout: %w", err)
		}

		ack := &types.Ack{
			Opcode:   types.OpCodeACK,
			BlockNum: data.BlockNum,
		}

		b, errM := ack.MarshalBinary()
		if errM != nil {
			return wrongBlockNum, nullBytes, fmt.Errorf("error while marshlling ack: %w", errM)
		}

		_, errW := c.conn.Write(b)
		if errW == nil {
			return data.BlockNum, uint16(copied), nil
		}

		c.l.Errorf("error while writing data packet: %s", errW.Error())

		continue
	}

	return wrongBlockNum, nullBytes, utils.ErrPacketCanNotBeSent
}

func (c *Connection) Receive(file string) error {
	errPacket := notDefinedError()

	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		c.l.Errorf("error while opening file: %s", err.Error())

		return sendErrorPacket(c.conn, errPacket)
	}

	defer func() {
		if err := f.Close(); err != nil {
			c.l.Errorf("error while closing file: %s", err.Error())
		}
	}()

	block := make([]byte, 0, types.MaxPayloadSize)
	blockBuffer := bytes.NewBuffer(block)
	var bytesAccum uint16

	for {
		blockNum, n, err := c.ReceiveBlock(blockBuffer)
		if err != nil {
			if errors.Is(err, utils.ErrPacketCanNotBeSent) {
				return err
			}

			errPacket = &types.Error{
				Opcode:    types.OpCodeError,
				ErrorCode: types.ErrNotDefined,
				ErrMsg:    "server can not create data packet",
			}

			return sendErrorPacket(c.conn, errPacket)
		}

		_, errW := f.Write(blockBuffer.Bytes())
		if errW != nil {
			return errors.New("error while writing block to file")
		}

		if c.trace {
			c.l.Debugf("received block#=%d, received #bytes=%d", blockNum, len(blockBuffer.Bytes()))
		}

		blockBuffer.Reset()
		bytesAccum += n

		if n < types.MaxPayloadSize {
			c.l.Debugf("received %d blocks, received %d bytes", blockNum, bytesAccum)
			return nil
		}
	}
}

func (c *Connection) SendBlock(block []byte, blockNum uint16) error {
	var ack types.Ack
	var errPacket types.Error

	data := &types.Data{
		Opcode:   types.OpCodeDATA,
		Payload:  block,
		BlockNum: blockNum,
	}
	buffer := make([]byte, types.DatagramSize)

	b, err := data.MarshalBinary()
	if err != nil {
		return fmt.Errorf("error while marshalling data packet: %w", err)
	}

	for i := c.numTries; i > 0; i-- {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return fmt.Errorf("error while setting write timeout: %w", err)
		}

		if _, err := c.conn.Write(b); err != nil {
			c.l.Errorf("error while writing data packet: %s", err.Error())

			continue
		}

		if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return fmt.Errorf("error while setting read timeout: %w", err)
		}

		n, err := c.conn.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			c.l.Errorf("error while reading response: %s", err.Error())

			continue
		}

		switch {
		case ack.UnmarshalBinary(buffer[:n]) == nil:
			if ack.BlockNum != blockNum {
				c.l.Errorf("ack block# %d != expected block# %d", ack.BlockNum, blockNum)

				continue
			}

			return nil
		case errPacket.UnmarshalBinary(buffer[:n]) == nil:
			return utils.ErrPacketCanNotBeSent
		default:
			continue
		}
	}

	return utils.ErrPacketCanNotBeSent
}

func (c *Connection) Send(file string) error {
	errPacket := notDefinedError()

	f, errOpen := os.Open(file)
	if errOpen != nil {
		c.l.Errorf("error while opening file: %s", errOpen.Error())

		return sendErrorPacket(c.conn, errPacket)
	}

	defer func() {
		if err := f.Close(); err != nil {
			c.l.Errorf("error while closing file: %s", err.Error())
		}
	}()

	var blockNum uint16 = 1

	block := make([]byte, types.MaxPayloadSize)
	bytesAccum := 0

	for {
		n, err := f.Read(block)
		if err != nil {
			c.l.Errorf("error while reading file block: %s", err.Error())

			return sendErrorPacket(c.conn, errPacket)
		}

		if err := c.SendBlock(block[:n], blockNum); err != nil {
			c.l.Errorf("error while sending data packet: %s", err.Error())

			errPacket = &types.Error{
				Opcode:    types.OpCodeError,
				ErrorCode: types.ErrNotDefined,
				ErrMsg:    "server can not create data packet",
			}

			return sendErrorPacket(c.conn, errPacket)
		}

		if c.trace {
			c.l.Debugf("sent block#=%d, sent #bytes=%d", blockNum, n)
		}

		blockNum++
		bytesAccum += n

		if n < types.MaxPayloadSize {
			c.l.Debugf("sent %d blocks, sent %d bytes", blockNum, bytesAccum)

			return nil
		}
	}
}
