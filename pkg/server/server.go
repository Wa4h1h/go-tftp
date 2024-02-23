package server

import (
	"errors"
	"fmt"
	"github.com/Wa4h1h/go-tftp/pkg/types"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
	"go.uber.org/zap"
	"net"
	"time"
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
	writeTimeout uint, numTries int, tftpFolder string) *Server {
	return &Server{logger: l, port: port,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		numTries:     numTries,
		tftpFolder:   tftpFolder,
	}
}

func (s *Server) ListenAndServe() error {
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		return utils.ErrStartingServer
	}

	s.conn = conn

	for {
		datagram := make([]byte, types.DatagramSize)

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
	conn, err := net.Dial("udp", addr.String())
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
		s.logger.Errorf("error while reading request")

		return
	}

	switch req.Opcode {
	case types.OpCodeRRQ:
		sender := NewSender(conn, s.logger,
			time.Duration(s.readTimeout)*time.Second,
			time.Duration(s.writeTimeout)*time.Second,
			s.numTries)

		err := sender.send(fmt.Sprintf("%s/%s", s.tftpFolder, req.Filename))
		if err != nil {
			s.logger.Errorf("error while responding to rrq: %s", err.Error())

			return
		}
	case types.OpCodeWRQ:
		receiver := NewReceiver(conn, s.logger,
			time.Duration(s.readTimeout)*time.Second,
			time.Duration(s.writeTimeout)*time.Second,
			s.numTries)

		err := receiver.receive(fmt.Sprintf("%s/%s", s.tftpFolder, req.Filename))
		if err != nil {
			s.logger.Errorf("error while responding to wrq: %s", err.Error())

			return
		}
	default:
		unknownOp := &types.Error{
			Opcode:    types.OpCodeError,
			ErrorCode: types.ErrIllegalTftpOp,
			ErrMsg:    fmt.Sprintf("server can not resolve request operation code %d", req.Opcode),
		}
		if err := sendErrorPacket(conn, unknownOp); err != nil {
			s.logger.Errorf("error while responding to wrq: %s", err.Error())

			return
		}
	}
}
