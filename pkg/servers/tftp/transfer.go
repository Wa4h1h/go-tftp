package tftp

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/WadhahJemai/go-tftp/pkg/types"
	"github.com/WadhahJemai/go-tftp/pkg/utils"
	"go.uber.org/zap"
	"io"
	"net"
	"os"
	"time"
)

type Transfer interface {
	send(file string) error
	sendBlock(block []byte, blockNum uint16) error
	acknowledgeWrq() error
	receive(file string) error
	receiveBlock(blockW io.Writer) (uint16, uint16, error)
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

		return utils.ErrPacketMarshall
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
				if errors.Is(err, io.EOF) {
					return utils.ErrOtherSideConnClosed
				}
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
				return utils.ErrPacketCanNotBeSent
			default:
				continue
			}

			c.l.Debug(fmt.Sprintf("received ack block#=%d", ack.BlockNum))
		}

		return nil
	}

	return utils.ErrPacketCanNotBeSent
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
			ErrMsg:    "file too large to be transferred over tftp",
		}

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
			if errors.Is(err, utils.ErrPacketCanNotBeSent) || errors.Is(err, utils.ErrOtherSideConnClosed) {
				return err
			}

			errPacket = &types.Error{
				Opcode:    types.OpCodeError,
				ErrorCode: types.ErrNotDefined,
				ErrMsg:    "server can not create data packet",
			}

			return sendErrorPacket(c.conn, errPacket)
		}

		c.l.Debug(fmt.Sprintf("sent block#=%d, sent #bytes=%d", blockNum, n))

		blockNum++

		if n < types.MaxPayloadSize {
			return nil
		}
	}
}

func (c *Connection) acknowledgeWrq() error {
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
		c.l.Error(fmt.Sprintf("error while setting write timeout: %s", err.Error()))

		return utils.ErrCanNotSetWriteTimeout
	}

	_, errW := c.conn.Write(b)
	if errW != nil {
		c.l.Error(fmt.Sprintf("error while writing data packet: %s", errW.Error()))

		return utils.ErrPacketCanNotBeSent
	}

	return nil
}

func (c *Connection) receiveBlock(blockW io.Writer) (uint16, uint16, error) {
	var data types.Data
	var receivedBlockNum uint16
	var nullBytes uint16

	datagram := make([]byte, types.DatagramSize)
	for i := c.numTries; i > 0; i-- {
		if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			c.l.Error(fmt.Sprintf("error while setting read timeout: %s", err.Error()))

			return receivedBlockNum, nullBytes, utils.ErrCanNotSetReadTimeout
		}

		n, err := c.conn.Read(datagram)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return receivedBlockNum, nullBytes, utils.ErrOtherSideConnClosed
			}
			c.l.Error(fmt.Sprintf("error while reading ack: %s", err.Error()))

			continue
		}

		if n < 0 {
			c.l.Debug("read 0 bytes")

			continue
		}

		if err := data.UnmarshalBinary(datagram[:n]); err != nil {
			c.l.Error(fmt.Sprintf("error while unmarshal data packet: %s", err.Error()))

			continue
		}

		src := bytes.NewBuffer(data.Payload)
		copied, errCopy := io.CopyN(blockW, src, int64(len(data.Payload)))
		if errCopy != nil {
			c.l.Error(fmt.Sprintf("error while copy payload: %s", err.Error()))

			return receivedBlockNum, nullBytes, utils.ErrCanNotCopySLice
		}

		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			c.l.Error(fmt.Sprintf("error while setting write timeout: %s", err.Error()))

			return receivedBlockNum, nullBytes, utils.ErrCanNotSetWriteTimeout
		}

		ack := &types.Ack{
			Opcode:   types.OpCodeACK,
			BlockNum: data.BlockNum,
		}

		b, errM := ack.MarshalBinary()
		if errM != nil {
			c.l.Error(err.Error())

			return receivedBlockNum, nullBytes, utils.ErrPacketMarshall
		}

		_, errW := c.conn.Write(b)
		if errW == nil {
			return data.BlockNum, uint16(copied), nil
		}

		c.l.Error(fmt.Sprintf("error while writing data packet: %s", err.Error()))

		continue

	}

	return receivedBlockNum, nullBytes, utils.ErrPacketCanNotBeSent
}

func (c *Connection) receive(file string) error {
	errPacket := notDefinedError()

	var errStat error

	_, errStat = os.Stat(file)

	switch {
	case errStat == nil:
		errPacket = &types.Error{
			Opcode:    types.OpCodeError,
			ErrorCode: types.ErrFileAlreadyExists,
			ErrMsg:    fmt.Sprintf("%s already exists", file),
		}

		return sendErrorPacket(c.conn, errPacket)
	case errStat != nil && !os.IsNotExist(errStat):
		c.l.Error(fmt.Sprintf("error while checking file exists: %s", errStat.Error()))

		return sendErrorPacket(c.conn, errPacket)
	}

	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		c.l.Error(fmt.Sprintf("error while opening file: %s", err.Error()))

		return sendErrorPacket(c.conn, errPacket)
	}

	defer func() {
		if err := f.Close(); err != nil {
			c.l.Error(fmt.Sprintf("error while closing file: %s", err.Error()))
		}
	}()

	if err := c.acknowledgeWrq(); err != nil {
		return err
	}

	block := make([]byte, 0, types.MaxPayloadSize)
	for {
		blockBuffer := bytes.NewBuffer(block)

		blockNum, n, err := c.receiveBlock(blockBuffer)
		if err != nil {
			if errors.Is(err, utils.ErrPacketCanNotBeSent) || errors.Is(err, utils.ErrOtherSideConnClosed) {
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

		c.l.Debug(fmt.Sprintf("received block#=%d, received #bytes=%d", blockNum, len(blockBuffer.Bytes())))

		blockBuffer.Reset()

		if n < types.MaxPayloadSize {
			return nil
		}
	}
}
