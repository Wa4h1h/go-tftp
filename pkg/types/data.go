package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
)

type Data struct {
	Payload  []byte
	BlockNum uint16
	Opcode   OpCode
}

func (d *Data) MarshalBinary() ([]byte, error) {
	if len(d.Payload) > MaxPayloadSize {
		return nil, utils.ErrDataPayloadTooBig
	}

	b := new(bytes.Buffer)
	dataLen := 2 + 2 + len(d.Payload)
	b.Grow(dataLen)

	if err := binary.Write(b, binary.BigEndian, &d.Opcode); err != nil {
		return nil, fmt.Errorf("error while writing opcode: %w", err)
	}

	if err := binary.Write(b, binary.BigEndian, &d.BlockNum); err != nil {
		return nil, fmt.Errorf("error while writing block#: %w", err)
	}

	if _, err := b.Write(d.Payload); err != nil {
		return nil, fmt.Errorf("error while writing payload: %w", err)
	}

	return b.Bytes(), nil
}

func (d *Data) UnmarshalBinary(data []byte) error {
	b := bytes.NewBuffer(data)

	if err := binary.Read(b, binary.BigEndian, &d.Opcode); err != nil {
		return fmt.Errorf("error while reading opcode: %w", err)
	}

	if d.Opcode != OpCodeDATA {
		return utils.ErrWrongOpCode
	}

	if err := binary.Read(b, binary.BigEndian, &d.BlockNum); err != nil {
		return fmt.Errorf("error while reading block#: %w", err)
	}

	d.Payload = data[4:]

	return nil
}
