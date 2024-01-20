package tftp

import (
	"fmt"
	"github.com/WadhahJemai/go-tftp/internal/types"
	"net"
)

func NotDefinedError() *types.Error {
	return &types.Error{
		Opcode:    types.OpCodeError,
		ErrorCode: types.ErrNotDefined,
		ErrMsg:    "no defined error",
	}
}

func SendErrorPacket(conn net.Conn, errorPacket *types.Error) error {
	b, errM := errorPacket.MarshalBinary()
	if errM != nil {
		return fmt.Errorf("error while marshal error packet: %s", errM)
	}

	if _, err := conn.Write(b); err != nil {
		return fmt.Errorf("error while marshal error packet: %s", err)
	}

	return nil
}
