package tftp

import (
	"errors"
	"fmt"
	"github.com/WadhahJemai/go-tftp/internal/utils"
	"go.uber.org/zap"
	"net"
	"os"
	"time"
)

type Server struct {
	port         string
	readTimeout  uint
	writeTimeout uint
	logger       *zap.Logger
	conn         net.PacketConn
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

	for {
		datagram := make([]byte, 516)

		if err := conn.SetReadDeadline(time.Now().Add(time.Duration(s.readTimeout) * time.Second)); err != nil {
			s.logger.Error(fmt.Sprintf("setting read deadline failed: %s", err.Error()))

			return fmt.Errorf("error while setting reading deadline")
		}

		n, addr, err := conn.ReadFrom(datagram)
		if err != nil && !errors.Is(err, net.ErrClosed) {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				s.logger.Error("reading timed out")
			}

			return nil
		}

		go s.handle(n, addr, conn)
	}
}

func (s *Server) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("error while closing connection: %w", err)
	}

	return nil
}

func (s *Server) handle(n int, remoteAddr net.Addr, conn net.PacketConn) {
	conn.WriteTo([]byte("hello world"), remoteAddr)
}
