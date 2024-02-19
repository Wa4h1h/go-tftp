package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
)

type Ack struct {
	Opcode   OpCode
	BlockNum uint16
}

func (a *Ack) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	ackLength := 2 + 2
	b.Grow(ackLength)

	if err := binary.Write(b, binary.BigEndian, &a.Opcode); err != nil {
		return nil, fmt.Errorf("error while writing opcode: %w", err)
	}

	if err := binary.Write(b, binary.BigEndian, &a.BlockNum); err != nil {
		return nil, fmt.Errorf("error while writing block#: %w", err)
	}

	return b.Bytes(), nil
}

func (a *Ack) UnmarshalBinary(data []byte) error {
	b := bytes.NewBuffer(data)

	if err := binary.Read(b, binary.BigEndian, &a.Opcode); err != nil {
		return fmt.Errorf("error while reading opcode: %w", err)
	}

	if a.Opcode != OpCodeACK {
		return utils.ErrWrongOpCode
	}

	if err := binary.Read(b, binary.BigEndian, &a.BlockNum); err != nil {
		return fmt.Errorf("error while reading block#: %w", err)
	}

	return nil
}
