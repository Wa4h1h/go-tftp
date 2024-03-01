package server

import (
	"fmt"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"syscall"

	"go.uber.org/zap"

	"github.com/Wa4h1h/go-tftp/pkg/types"
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

func reusePort() control {
	return func(network, address string, c syscall.RawConn) error {
		var opErr error

		err := c.Control(func(fd uintptr) {
			opErr = syscall.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		})
		if err != nil {
			opErr = err
		}

		return opErr
	}
}

func assertSenderFile(l *zap.SugaredLogger, c net.Conn, filename string) (bool, error) {
	errPacket := notDefinedError()

	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			errPacket = &types.Error{
				Opcode:    types.OpCodeError,
				ErrorCode: types.ErrFileNotFound,
				ErrMsg:    fmt.Sprintf("%s not found", filename),
			}
		} else {
			l.Errorf("error while checking file exists: %s", err.Error())
		}

		return false, sendErrorPacket(c, errPacket)
	}

	return true, nil
}

func assertReceiverFile(l *zap.SugaredLogger, c net.Conn, filename string) (bool, error) {
	errPacket := notDefinedError()
	_, err := os.Stat(filename)

	switch {
	case err == nil:
		errPacket = &types.Error{
			Opcode:    types.OpCodeError,
			ErrorCode: types.ErrFileAlreadyExists,
			ErrMsg:    fmt.Sprintf("%s already exists", filename),
		}

		return false, sendErrorPacket(c, errPacket)
	case !os.IsNotExist(err):
		l.Errorf("error while checking file exists: %s", err.Error())

		return false, sendErrorPacket(c, errPacket)
	}

	return true, nil
}
