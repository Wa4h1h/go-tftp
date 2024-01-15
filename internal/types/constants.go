package types

type OpCode uint16

const (
	OpCodeRRQ OpCode = iota + 1
	OpCodeWRQ
	OpCodeDATA
	OpCodeACK
	OpCodeError
)
