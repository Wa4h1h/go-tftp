package tftp

import (
	"errors"
	"fmt"
	"github.com/WadhahJemai/go-tftp/internal/types"
	"github.com/WadhahJemai/go-tftp/internal/utils"
	"go.uber.org/zap"
	"net"
	"os"
	"time"
)

type Transfer interface {
	send(file string) error
	sendBlock(block []byte, blockNum uint16) error
	receive(file string) error
	receiveBlock(block []byte, blockNum uint16) error
}

type Connection struct {
	conn         net.Conn
	l            *zap.Logger
	numTries     int
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewConnection(conn net.Conn,
	logger *zap.Logger, readTimeout time.Duration,
	writeTimeout time.Duration, numTries int) *Connection {
	return &Connection{conn: conn, l: logger, readTimeout: readTimeout, writeTimeout: writeTimeout, numTries: numTries}
}

func (c *Connection) sendBlock(block []byte, blockNum uint16) error {
	var ack types.Ack
	var errPacket types.Error

	data := &types.Data{
		Opcode:   types.OpCodeDATA,
		Payload:  block,
		BlockNum: blockNum,
	}

	b, err := data.MarshalBinary()
	if err != nil {
		c.l.Error(err.Error())

		return utils.ErrDataMarshall
	}

	for i := c.numTries; i > 0; i-- {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			c.l.Error(fmt.Sprintf("error while setting write timeout: %s", err.Error()))

			return utils.ErrCanNotSetWriteTimeout
		}

		if _, err := c.conn.Write(b); err != nil {
			c.l.Error(fmt.Sprintf("error while writing data packet: %s", err.Error()))

			continue
		}

		if !(len(block) < types.MaxPayloadSize) {
			if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
				c.l.Error(fmt.Sprintf("error while setting read timeout: %s", err.Error()))

				return utils.ErrCanNotSetReadTimeout
			}

			buf := make([]byte, types.DatagramSize)

			n, err := c.conn.Read(buf)
			if err != nil {
				c.l.Error(fmt.Sprintf("error while reading ack: %s", err.Error()))

				continue
			}

			switch {
			case ack.UnmarshalBinary(buf[:n]) == nil:
				if ack.BlockNum != blockNum {
					c.l.Error(fmt.Sprintf("ack block# %d != expected block# %d", ack.BlockNum, blockNum))

					continue
				}
			case errPacket.UnmarshalBinary(buf[:n]) == nil:
				return utils.ErrDataPacketCanNotBeSent
			default:
				continue
			}

			c.l.Debug(fmt.Sprintf("received ack block#=%d", ack.BlockNum))
		}

		return nil
	}

	return utils.ErrDataPacketCanNotBeSent
}

func (c *Connection) send(file string) error {
	errPacket := notDefinedError()

	stats, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			errPacket = &types.Error{
				Opcode:    types.OpCodeError,
				ErrorCode: types.ErrFileNotFound,
				ErrMsg:    fmt.Sprintf("%s not found", file),
			}
		} else {
			c.l.Error(fmt.Sprintf("error while checking file exists: %s", err.Error()))
		}

		return sendErrorPacket(c.conn, errPacket)
	}

	if stats.Size()/types.MaxPayloadSize > types.MaxBlocks {
		errPacket = &types.Error{
			Opcode:    types.OpCodeError,
			ErrorCode: types.ErrNotDefined,
			ErrMsg:    "file too large to be transferred over tftp"}

		return sendErrorPacket(c.conn, errPacket)
	}

	f, errOpen := os.Open(file)
	if errOpen != nil {
		c.l.Error(fmt.Sprintf("error while opening file: %s", errOpen.Error()))

		return sendErrorPacket(c.conn, errPacket)
	}

	defer func() {
		if err := f.Close(); err != nil {
			c.l.Error(fmt.Sprintf("error while closing file: %s", err.Error()))
		}
	}()

	var blockNum uint16 = 1

	for {
		block := make([]byte, types.MaxPayloadSize)

		n, err := f.Read(block)
		if err != nil {
			c.l.Error(fmt.Sprintf("error while reading file block: %s", err.Error()))

			return sendErrorPacket(c.conn, errPacket)
		}

		if err := c.sendBlock(block[:n], blockNum); err != nil {
			if errors.Is(err, utils.ErrDataPacketCanNotBeSent) {
				return err
			} else if errors.Is(err, utils.ErrDataMarshall) {
				errPacket = &types.Error{
					Opcode:    types.OpCodeError,
					ErrorCode: types.ErrNotDefined,
					ErrMsg:    "server can not create data packet",
				}

				return sendErrorPacket(c.conn, errPacket)
			}
		}

		c.l.Debug(fmt.Sprintf("sent block#=%d, sent #bytes=%d", blockNum, n))

		blockNum++

		if n < types.MaxPayloadSize {
			return nil
		}
	}
}

func (c *Connection) receiveBlock(block []byte, blockNum uint16) error {
	return nil
}

func (c *Connection) receive(file string) error {
	return nil
}
