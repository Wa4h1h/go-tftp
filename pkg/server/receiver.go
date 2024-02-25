package server

import (
	"bytes"
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

type Receiver interface {
	AcknowledgeWrq() error
	Receive(file string) error
	ReceiveBlock(blockW io.Writer) (uint16, uint16, error)
}

type Incoming struct {
	conn         net.Conn
	l            *zap.SugaredLogger
	numTries     int
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewReceiver(conn net.Conn,
	logger *zap.SugaredLogger, readTimeout time.Duration,
	writeTimeout time.Duration, numTries int) Receiver {
	return &Incoming{conn: conn, l: logger, readTimeout: readTimeout, writeTimeout: writeTimeout, numTries: numTries}
}

func (i *Incoming) AcknowledgeWrq() error {
	ack := &types.Ack{
		Opcode:   types.OpCodeACK,
		BlockNum: 0,
	}

	b, err := ack.MarshalBinary()
	if err != nil {
		i.l.Error(err.Error())

		return utils.ErrPacketMarshall
	}

	if err := i.conn.SetWriteDeadline(time.Now().Add(i.writeTimeout)); err != nil {
		i.l.Errorf("error while setting write timeout: %s", err.Error())

		return utils.ErrCanNotSetWriteTimeout
	}

	if _, err := i.conn.Write(b); err != nil {
		i.l.Errorf("error while writing data packet: %s", err.Error())

		return utils.ErrPacketCanNotBeSent
	}

	return nil
}

func (i *Incoming) ReceiveBlock(blockW io.Writer) (uint16, uint16, error) {
	var data types.Data
	var wrongBlockNum uint16
	var nullBytes uint16

	datagram := make([]byte, types.DatagramSize)

	for tries := i.numTries; tries > 0; tries-- {
		if err := i.conn.SetReadDeadline(time.Now().Add(i.readTimeout)); err != nil {
			i.l.Errorf("error while setting read timeout: %s", err.Error())

			return wrongBlockNum, nullBytes, utils.ErrCanNotSetReadTimeout
		}

		n, err := i.conn.Read(datagram)
		if err != nil && !errors.Is(err, io.EOF) {
			i.l.Errorf("error while reading ack: %s", err.Error())

			continue
		}

		if n < 0 {
			i.l.Debugf("read 0 bytes")

			continue
		}

		if err := data.UnmarshalBinary(datagram[:n]); err != nil {
			i.l.Errorf("error while unmarshal data packet: %s", err.Error())

			continue
		}

		src := bytes.NewBuffer(data.Payload)

		copied, errCopy := io.CopyN(blockW, src, int64(len(data.Payload)))
		if errCopy != nil {
			i.l.Errorf("error while copy payload: %s", errCopy.Error())

			return wrongBlockNum, nullBytes, utils.ErrCanNotCopySLice
		}

		i.l.Infof("received --> %v", data)

		if err := i.conn.SetWriteDeadline(time.Now().Add(i.writeTimeout)); err != nil {
			i.l.Errorf("error while setting write timeout: %s", err.Error())

			return wrongBlockNum, nullBytes, utils.ErrCanNotSetWriteTimeout
		}

		ack := &types.Ack{
			Opcode:   types.OpCodeACK,
			BlockNum: data.BlockNum,
		}

		b, errM := ack.MarshalBinary()
		if errM != nil {
			i.l.Error(errM.Error())

			return wrongBlockNum, nullBytes, utils.ErrPacketMarshall
		}

		_, errW := i.conn.Write(b)
		if errW == nil {
			return data.BlockNum, uint16(copied), nil
		}

		i.l.Errorf("error while writing data packet: %s", errW.Error())

		continue
	}

	return wrongBlockNum, nullBytes, utils.ErrPacketCanNotBeSent
}

func (i *Incoming) Receive(file string) error {
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

		return sendErrorPacket(i.conn, errPacket)
	case !os.IsNotExist(errStat):
		i.l.Errorf("error while checking file exists: %s", errStat.Error())

		return sendErrorPacket(i.conn, errPacket)
	}

	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		i.l.Errorf("error while opening file: %s", err.Error())

		return sendErrorPacket(i.conn, errPacket)
	}

	defer func() {
		if err := f.Close(); err != nil {
			i.l.Errorf("error while closing file: %s", err.Error())
		}
	}()

	block := make([]byte, 0, types.MaxPayloadSize)
	blockBuffer := bytes.NewBuffer(block)

	for {
		blockNum, n, err := i.ReceiveBlock(blockBuffer)
		if err != nil {
			if errors.Is(err, utils.ErrPacketCanNotBeSent) {
				return err
			}

			errPacket = &types.Error{
				Opcode:    types.OpCodeError,
				ErrorCode: types.ErrNotDefined,
				ErrMsg:    "server can not create data packet",
			}

			return sendErrorPacket(i.conn, errPacket)
		}

		_, errW := f.Write(blockBuffer.Bytes())
		if errW != nil {
			return errors.New("error while writing block to file")
		}

		i.l.Debugf("received block#=%d, received #bytes=%d", blockNum, len(blockBuffer.Bytes()))

		blockBuffer.Reset()

		if n < types.MaxPayloadSize {
			return nil
		}
	}
}
