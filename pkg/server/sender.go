package server

import (
	"errors"
	"fmt"
	"github.com/Wa4h1h/go-tftp/pkg/types"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
	"go.uber.org/zap"
	"io"
	"net"
	"os"
	"time"
)

type Sender interface {
	Send(file string) error
	SendBlock(block []byte, blockNum uint16) error
}

type Outgoing struct {
	conn         net.Conn
	l            *zap.SugaredLogger
	numTries     int
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewSender(conn net.Conn,
	logger *zap.SugaredLogger, readTimeout time.Duration,
	writeTimeout time.Duration, numTries int) Sender {
	return &Outgoing{conn: conn, l: logger, readTimeout: readTimeout, writeTimeout: writeTimeout, numTries: numTries}
}

func (o *Outgoing) SendBlock(block []byte, blockNum uint16) error {
	var ack types.Ack
	var errPacket types.Error

	data := &types.Data{
		Opcode:   types.OpCodeDATA,
		Payload:  block,
		BlockNum: blockNum,
	}

	b, err := data.MarshalBinary()
	if err != nil {
		o.l.Error(err.Error())

		return utils.ErrPacketMarshall
	}

	for i := o.numTries; i > 0; i-- {
		if err := o.conn.SetWriteDeadline(time.Now().Add(o.writeTimeout)); err != nil {
			o.l.Errorf("error while setting write timeout: %s", err.Error())

			return utils.ErrCanNotSetWriteTimeout
		}

		if _, err := o.conn.Write(b); err != nil {
			o.l.Errorf("error while writing data packet: %s", err.Error())

			continue
		}

		if !(len(block) < types.MaxPayloadSize) {
			if err := o.conn.SetReadDeadline(time.Now().Add(o.readTimeout)); err != nil {
				o.l.Errorf("error while setting read timeout: %s", err.Error())

				return utils.ErrCanNotSetReadTimeout
			}

			buf := make([]byte, types.DatagramSize)

			n, err := o.conn.Read(buf)
			if err != nil && !errors.Is(err, io.EOF) {
				o.l.Errorf("error while reading ack: %s", err.Error())

				continue
			}

			switch {
			case ack.UnmarshalBinary(buf[:n]) == nil:
				if ack.BlockNum != blockNum {
					o.l.Errorf("ack block# %d != expected block# %d", ack.BlockNum, blockNum)

					continue
				}
			case errPacket.UnmarshalBinary(buf[:n]) == nil:
				return utils.ErrPacketCanNotBeSent
			default:
				continue
			}

			o.l.Debugf("received ack block#=%d", ack.BlockNum)
		}

		return nil
	}

	return utils.ErrPacketCanNotBeSent
}

func (o *Outgoing) Send(file string) error {
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
			o.l.Errorf("error while checking file exists: %s", err.Error())
		}

		return sendErrorPacket(o.conn, errPacket)
	}

	if stats.Size()/types.MaxPayloadSize > types.MaxBlocks {
		errPacket = &types.Error{
			Opcode:    types.OpCodeError,
			ErrorCode: types.ErrNotDefined,
			ErrMsg:    "file too large to be transferred over tftp",
		}

		return sendErrorPacket(o.conn, errPacket)
	}

	f, errOpen := os.Open(file)
	if errOpen != nil {
		o.l.Errorf("error while opening file: %s", errOpen.Error())

		return sendErrorPacket(o.conn, errPacket)
	}

	defer func() {
		if err := f.Close(); err != nil {
			o.l.Errorf("error while closing file: %s", err.Error())
		}
	}()

	var blockNum uint16 = 1

	for {
		block := make([]byte, types.MaxPayloadSize)

		n, err := f.Read(block)
		if err != nil {
			o.l.Errorf("error while reading file block: %s", err.Error())

			return sendErrorPacket(o.conn, errPacket)
		}

		if err := o.SendBlock(block[:n], blockNum); err != nil {
			if errors.Is(err, utils.ErrPacketCanNotBeSent) {
				return err
			}

			errPacket = &types.Error{
				Opcode:    types.OpCodeError,
				ErrorCode: types.ErrNotDefined,
				ErrMsg:    "server can not create data packet",
			}

			return sendErrorPacket(o.conn, errPacket)
		}

		o.l.Debugf("sent block#=%d, sent #bytes=%d", blockNum, n)

		blockNum++

		if n < types.MaxPayloadSize {
			return nil
		}
	}
}
