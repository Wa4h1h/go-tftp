package tftp

import (
	"errors"
	"fmt"
	"github.com/WadhahJemai/go-tftp/internal/types"
	"github.com/WadhahJemai/go-tftp/internal/utils"
	"go.uber.org/zap"
	"net"
)

type Server struct {
	port         string
	logger       *zap.Logger
	conn         net.PacketConn
	readTimeout  uint
	writeTimeout uint
}

func NewServer(l *zap.Logger, port string, readTimeout uint, writeTimeout uint) *Server {
	return &Server{logger: l, port: port, readTimeout: readTimeout, writeTimeout: writeTimeout}
}

func (s *Server) ListenAndServe() error {
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		return utils.ErrStartingServer
	}

	s.conn = conn
	var req types.Request
	for {
		datagram := make([]byte, 516)
		n, addr, err := conn.ReadFrom(datagram)
		if err != nil && !errors.Is(err, net.ErrClosed) {
			return nil
		}

		if err := req.UnmarshalBinary(datagram); err != nil {
			continue
		}

		go s.handle(n, addr, datagram)
	}
}

func (s *Server) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("error while closing connection: %w", err)
	}

	return nil
}

func (s *Server) handle(n int, remoteAddr net.Addr, datagram []byte) {
	s.logger.Info(string(datagram))
}
