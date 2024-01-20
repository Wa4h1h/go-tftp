package tftp

import (
	"fmt"
	"github.com/WadhahJemai/go-tftp/internal/types"
	"github.com/WadhahJemai/go-tftp/internal/utils"
	"go.uber.org/zap"
	"net"
	"os"
	"time"
)

type FileSender interface {
	send(file string) error
	sendBlock(block []byte, blockNum uint16) error
}

type Sender struct {
	conn         net.Conn
	l            *zap.Logger
	numTries     int
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewTSender(conn net.Conn, logger *zap.Logger, readTimeout time.Duration, writeTimeout time.Duration, numTries int) *Sender {
	return &Sender{conn: conn, l: logger, readTimeout: readTimeout, writeTimeout: writeTimeout, numTries: numTries}
}

func (s *Sender) sendBlock(block []byte, blockNum uint16) error {
	s.l.Debug(fmt.Sprintf("block# ---> %d", blockNum))
	var ack types.Ack

	data := &types.Data{
		Opcode:   types.OpCodeDATA,
		Payload:  block,
		BlockNum: blockNum,
	}

	b, err := data.MarshalBinary()
	if err != nil {
		s.l.Error(fmt.Sprintf("%s", err.Error()))

		return utils.ErrDataMarshall
	}

	for i := s.numTries; i > 0; i-- {
		if err := s.conn.SetWriteDeadline(time.Now().Add(s.writeTimeout)); err != nil {
			s.l.Error(fmt.Sprintf("error while setting write timeout: %s", err.Error()))

			continue
		}

		if _, err := s.conn.Write(b); err != nil {
			s.l.Error(fmt.Sprintf("error while writing data packet: %s", err.Error()))

			continue
		}

		if !(len(block) < types.MaxPayloadSize) {
			if err := s.conn.SetReadDeadline(time.Now().Add(s.readTimeout)); err != nil {
				s.l.Error(fmt.Sprintf("error while setting read timeout: %s", err.Error()))

				continue
			}
			ackBytes := make([]byte, 4)
			if _, err := s.conn.Read(ackBytes); err != nil {
				s.l.Error(fmt.Sprintf("error while reading ack: %s", err.Error()))

				continue
			}

			if err := ack.UnmarshalBinary(ackBytes); err != nil {
				s.l.Error(fmt.Sprintf("error while unmashall ack: %s", err.Error()))

				continue
			}

			if ack.BlockNum != blockNum {
				s.l.Error(fmt.Sprintf("ack block# %d != expected block# %d", ack.BlockNum, blockNum))

				continue
			} else {
				s.l.Debug(fmt.Sprintf("received block#=%d", ack.BlockNum))

				return nil
			}
		}

		return nil
	}

	return utils.ErrDataPacketCanNotBeSent
}

func (s *Sender) send(file string) *types.Error {
	stats, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.Error{Opcode: types.OpCodeError,
				ErrorCode: types.ErrFileNotFound,
				ErrMsg:    fmt.Sprintf("%s not found", file)}
		}

		s.l.Error(fmt.Sprintf("error while checking file exists: %s", err.Error()))

		return NotDefinedError()
	}

	if stats.Size()/types.MaxPayloadSize > types.MaxBlocks {
		return &types.Error{Opcode: types.OpCodeError,
			ErrorCode: types.ErrNotDefined,
			ErrMsg:    "file too large to be transferred over tftp"}
	}

	f, errOpen := os.Open(file)
	if errOpen != nil {
		s.l.Error(fmt.Sprintf("error while opening file: %s", errOpen.Error()))

		return NotDefinedError()
	}

	defer func() {
		if err := f.Close(); err != nil {
			s.l.Error(fmt.Sprintf("error while closing file: %s", err.Error()))
		}
	}()

	var blockNum uint16 = 1
	for {
		block := make([]byte, types.MaxPayloadSize)
		n, err := f.Read(block)
		if err != nil {
			s.l.Error("error while reading block")

			return NotDefinedError()
		}

		if err := s.sendBlock(block[:n], blockNum); err != nil {
			s.l.Error(fmt.Sprintf("error while sending data packet: %s", err.Error()))

			return NotDefinedError()
		}

		s.l.Debug(fmt.Sprintf("sent block#=%d, sent #bytes=%d", blockNum, n))

		blockNum++

		if n < types.MaxPayloadSize {
			s.l.Debug(fmt.Sprintf("%d", n))
			return nil
		}
	}
}
