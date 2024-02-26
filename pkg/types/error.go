package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/Wa4h1h/go-tftp/pkg/utils"
)

type Error struct {
	ErrMsg    string
	ErrorCode ErrCode
	Opcode    OpCode
}

func (e *Error) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	errLength := 2 + 2 + len(e.ErrMsg) + 1
	b.Grow(errLength)

	if err := binary.Write(b, binary.BigEndian, &e.Opcode); err != nil {
		return nil, fmt.Errorf("error while writing opcode: %w", err)
	}

	if err := binary.Write(b, binary.BigEndian, &e.ErrorCode); err != nil {
		return nil, fmt.Errorf("error while writing error code: %w", err)
	}

	if _, err := b.WriteString(e.ErrMsg); err != nil {
		return nil, fmt.Errorf("error while writing error message: %w", err)
	}

	if err := b.WriteByte(0); err != nil {
		return nil, fmt.Errorf("error while writing null byte")
	}

	return b.Bytes(), nil
}

func (e *Error) UnmarshalBinary(data []byte) error {
	b := bytes.NewBuffer(data)
	var err error

	if err = binary.Read(b, binary.BigEndian, &e.Opcode); err != nil {
		return fmt.Errorf("error while reading opcode: %w", err)
	}

	if e.Opcode != OpCodeError {
		return utils.ErrWrongOpCode
	}

	if err = binary.Read(b, binary.BigEndian, &e.ErrorCode); err != nil {
		return fmt.Errorf("error while reading error code: %w", err)
	}

	e.ErrMsg, err = b.ReadString(0)
	if err != nil {
		return fmt.Errorf("error while reading error message: %w", err)
	}

	e.ErrMsg = strings.TrimRight(e.ErrMsg, string(byte(0)))

	return nil
}
