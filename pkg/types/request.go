package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/Wa4h1h/go-tftp/pkg/utils"
)

type Request struct {
	Filename string
	Mode     string
	Opcode   OpCode
}

func (r *Request) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	rqLen := 2 + len(r.Filename) + 1 + len(r.Mode) + 1

	b.Grow(rqLen)

	if err := binary.Write(b, binary.BigEndian, &r.Opcode); err != nil {
		return nil, fmt.Errorf("error while writing Opcode: %w", err)
	}

	if _, err := b.WriteString(r.Filename); err != nil {
		return nil, fmt.Errorf("error while writing filename: %w", err)
	}

	if err := b.WriteByte(0); err != nil {
		return nil, fmt.Errorf("error while writing null byte after filename: %w", err)
	}

	if _, err := b.WriteString(r.Mode); err != nil {
		return nil, fmt.Errorf("error while writing mode: %w", err)
	}

	if err := b.WriteByte(0); err != nil {
		return nil, fmt.Errorf("error while writing null byte after mode: %w", err)
	}

	return b.Bytes(), nil
}

func (r *Request) UnmarshalBinary(data []byte) error {
	var err error

	rd := bytes.NewBuffer(data)

	err = binary.Read(rd, binary.BigEndian, &r.Opcode)
	if err != nil {
		return fmt.Errorf("error while decoding opCode: %w", err)
	}

	if r.Opcode != OpCodeRRQ && r.Opcode != OpCodeWRQ {
		return utils.ErrWrongOpCode
	}

	r.Filename, err = rd.ReadString(0)
	if err != nil {
		return fmt.Errorf("error while decoding filename: %w", err)
	}

	r.Filename = strings.TrimRight(r.Filename, string(byte(0)))

	r.Mode, err = rd.ReadString(0)
	if err != nil {
		return fmt.Errorf("error while decoding mode: %w", err)
	}

	r.Mode = strings.TrimRight(r.Mode, string(byte(0)))

	return nil
}
