package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Wa4h1h/go-tftp/pkg/types"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
	"go.uber.org/zap"
)

type Server struct {
	port         string
	tftpFolder   string
	logger       *zap.SugaredLogger
	conn         net.PacketConn
	numTries     int
	readTimeout  uint
	writeTimeout uint
}

func NewServer(l *zap.SugaredLogger, port string, readTimeout uint,
	writeTimeout uint, numTries int, tftpFolder string,
) *Server {
	return &Server{
		logger: l, port: port,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		numTries:     numTries,
		tftpFolder:   tftpFolder,
	}
}

func (s *Server) ListenAndServe() error {
	l := net.ListenConfig{
		Control: reusePort(),
	}

	conn, err := l.ListenPacket(context.Background(), "udp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		s.logger.Error(err.Error())

		return utils.ErrStartingServer
	}

	s.conn = conn
	datagram := make([]byte, types.DatagramSize)

	for {
		n, addr, err := conn.ReadFrom(datagram)
		if err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}

		if n > 0 {
			go s.handlePacket(addr, datagram)
		}
	}
}

func (s *Server) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("error while closing connection: %w", err)
	}

	return nil
}

func (s *Server) handlePacket(addr net.Addr, datagram []byte) {
	d := net.Dialer{
		LocalAddr: s.conn.LocalAddr(),
		Control:   reusePort(),
	}

	conn, err := d.Dial("udp", addr.String())
	if err != nil {
		s.logger.Errorf(err.Error())

		return
	}

	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.Errorf("error while closing connection with %s: %s", conn.RemoteAddr().Network(), err.Error())
		}
	}()

	var req types.Request

	if err := req.UnmarshalBinary(datagram); err != nil {
		unknownOp := &types.Error{
			Opcode:    types.OpCodeError,
			ErrorCode: types.ErrIllegalTftpOp,
			ErrMsg:    fmt.Sprintf("server can not resolve request operation code %d", req.Opcode),
		}
		if err := sendErrorPacket(conn, unknownOp); err != nil {
			s.logger.Errorf("error while responding to request: %s", err.Error())
		}

		return
	}

	t := NewTransfer(conn, s.logger,
		time.Duration(s.readTimeout)*time.Second,
		time.Duration(s.writeTimeout)*time.Second,
		s.numTries)

	file := fmt.Sprintf("%s/%s", s.tftpFolder, req.Filename)

	switch req.Opcode {
	case types.OpCodeRRQ:
		{
			ok, err := assertSenderFile(s.logger, conn, file)
			s.logger.Info(ok, err)
			if ok && err == nil {
				if err := t.Send(file); err != nil {
					s.logger.Errorf("error while responding to rrq: %s", err.Error())
				}
			}
		}
	case types.OpCodeWRQ:
		{
			ok, err := assertReceiverFile(s.logger, conn, file)
			if ok && err == nil {
				if err := t.AcknowledgeWrq(); err != nil {
					s.logger.Errorf("error while acknowledging wrq: %s", err.Error())

					return
				}

				if err := t.Receive(file); err != nil {
					s.logger.Errorf("error while responding to wrq: %s", err.Error())
				}
			}
		}
	}
}
