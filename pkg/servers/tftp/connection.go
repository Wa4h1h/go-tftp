package tftp

import (
	"errors"
	"fmt"
	"github.com/WadhahJemai/go-tftp/pkg/types"
	"github.com/WadhahJemai/go-tftp/pkg/utils"
	"go.uber.org/zap"
	"net"
	"os"
	"time"
)

type Connection interface {
	send(file string) (*types.Error, error)
	sendBlock(block []byte, blockNum uint16) error
}

type Transfer struct {
	conn         net.Conn
	l            *zap.Logger
	numTries     int
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewTransfer(conn net.Conn,
	logger *zap.Logger, readTimeout time.Duration,
	writeTimeout time.Duration, numTries int) *Transfer {
	return &Transfer{conn: conn, l: logger, readTimeout: readTimeout, writeTimeout: writeTimeout, numTries: numTries}
}

func (t *Transfer) sendBlock(block []byte, blockNum uint16) error {
	var ack types.Ack
	var errPacket types.Error

	data := &types.Data{
		Opcode:   types.OpCodeDATA,
		Payload:  block,
		BlockNum: blockNum,
	}

	b, err := data.MarshalBinary()
	if err != nil {
		t.l.Error(err.Error())

		return utils.ErrDataMarshall
	}

	for i := t.numTries; i > 0; i-- {
		if err := t.conn.SetWriteDeadline(time.Now().Add(t.writeTimeout)); err != nil {
			t.l.Error(fmt.Sprintf("error while setting write timeout: %s", err.Error()))

			continue
		}

		if _, err := t.conn.Write(b); err != nil {
			t.l.Error(fmt.Sprintf("error while writing data packet: %s", err.Error()))

			continue
		}

		if !(len(block) < types.MaxPayloadSize) {
			if err := t.conn.SetReadDeadline(time.Now().Add(t.readTimeout)); err != nil {
				t.l.Error(fmt.Sprintf("error while setting read timeout: %s", err.Error()))

				continue
			}

			buf := make([]byte, types.DatagramSize)

			n, err := t.conn.Read(buf)
			if err != nil {
				t.l.Error(fmt.Sprintf("error while reading ack: %s", err.Error()))

				continue
			}

			switch {
			case ack.UnmarshalBinary(buf[:n]) == nil:
				if ack.BlockNum != blockNum {
					t.l.Error(fmt.Sprintf("ack block# %d != expected block# %d", ack.BlockNum, blockNum))

					continue
				}
			case errPacket.UnmarshalBinary(buf[:n]) == nil:
				return utils.ErrNonRecoverable
			default:
				continue
			}

			t.l.Debug(fmt.Sprintf("received ack block#=%d", ack.BlockNum))
		}

		return nil
	}

	return utils.ErrDataPacketCanNotBeSent
}

func (t *Transfer) send(file string) (*types.Error, error) {
	stats, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.Error{
				Opcode:    types.OpCodeError,
				ErrorCode: types.ErrFileNotFound,
				ErrMsg:    fmt.Sprintf("%s not found", file)}, nil
		}

		t.l.Error(fmt.Sprintf("error while checking file exists: %s", err.Error()))

		return notDefinedError(), nil
	}

	if stats.Size()/types.MaxPayloadSize > types.MaxBlocks {
		return &types.Error{
			Opcode:    types.OpCodeError,
			ErrorCode: types.ErrNotDefined,
			ErrMsg:    "file too large to be transferred over tftp"}, nil
	}

	f, errOpen := os.Open(file)
	if errOpen != nil {
		t.l.Error(fmt.Sprintf("error while opening file: %s", errOpen.Error()))

		return notDefinedError(), nil
	}

	defer func() {
		if err := f.Close(); err != nil {
			t.l.Error(fmt.Sprintf("error while closing file: %s", err.Error()))
		}
	}()

	var blockNum uint16 = 1

	for {
		block := make([]byte, types.MaxPayloadSize)

		n, err := f.Read(block)
		if err != nil {
			t.l.Error("error while reading block")

			return notDefinedError(), nil
		}

		if err := t.sendBlock(block[:n], blockNum); err != nil {
			if errors.Is(err, utils.ErrNonRecoverable) {
				return nil, err
			}

			t.l.Error(fmt.Sprintf("error while sending data packet: %s", err.Error()))

			return notDefinedError(), nil
		}

		t.l.Debug(fmt.Sprintf("sent block#=%d, sent #bytes=%d", blockNum, n))

		blockNum++

		if n < types.MaxPayloadSize {
			return nil, nil
		}
	}
}
