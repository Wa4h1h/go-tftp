package tftp

import (
	"errors"
	"fmt"
	"github.com/WadhahJemai/go-tftp/pkg/types"
	"github.com/WadhahJemai/go-tftp/pkg/utils"
	"go.uber.org/zap"
	"net"
	"time"
)

type Server struct {
	port         string
	tftpFolder   string
	logger       *zap.Logger
	conn         net.PacketConn
	numTries     int
	readTimeout  uint
	writeTimeout uint
}

func NewServer(l *zap.Logger, port string, readTimeout uint, writeTimeout uint, numTries int, tftpFolder string) *Server {
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
		s.logger.Error(err.Error())

		return
	}

	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.Error(fmt.Sprintf("error while closing connection with %s: %s", conn.RemoteAddr().Network(), err.Error()))
		}
	}()

	var req types.Request

	if err := req.UnmarshalBinary(datagram); err != nil {
		s.logger.Error("error while reading request")

		return
	}

	var c Transfer = NewConnection(conn, s.logger,
		time.Duration(s.readTimeout)*time.Second,
		time.Duration(s.writeTimeout)*time.Second,
		s.numTries)

	if req.Opcode == types.OpCodeRRQ {
		errPacket, err := c.send(fmt.Sprintf("%s/%s", s.tftpFolder, req.Filename))
		if err != nil {
			return
		}

		if errPacket != nil {
			if err := sendErrorPacket(conn, errPacket); err != nil {
				s.logger.Error(err.Error())

				return
			}
		}
	}
}
