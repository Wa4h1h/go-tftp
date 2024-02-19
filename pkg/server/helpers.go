package server

import (
	"fmt"
	"github.com/Wa4h1h/go-tftp/pkg/types"
	"net"
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
