package server

import (
	"fmt"
	"github.com/Wa4h1h/go-tftp/pkg/types"
	"net"
	"syscall"
)

func notDefinedError() *types.Error {
	return &types.Error{
		Opcode:    types.OpCodeError,
		ErrorCode: types.ErrNotDefined,
		ErrMsg:    "no defined error",
	}
}

func sendErrorPacket(conn net.Conn, errorPacket *types.Error) error {
	b, err := errorPacket.MarshalBinary()
	if err != nil {
		return fmt.Errorf("error while marshal error packet: %w", err)
	}

	if _, err := conn.Write(b); err != nil {
		return fmt.Errorf("error while marshal error packet: %w", err)
	}

	return nil
}

type control func(network, address string, c syscall.RawConn) error

func controlReusePort() control {
	return func(network, address string, c syscall.RawConn) error {
		var opErr error
		err := c.Control(func(fd uintptr) {
			opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
		})
		if err != nil {
			return err
		}
		return opErr
	}
}
