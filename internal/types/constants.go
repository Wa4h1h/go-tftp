package types

type OpCode uint16

const (
	OpCodeRRQ OpCode = iota + 1
	OpCodeWRQ
	OpCodeDATA
	OpCodeACK
	OpCodeError
)

type ErrCode uint16

const (
	ErrNotDefined ErrCode = iota
	ErrFileNotFound
	ErrAccessViolation
	ErrDiskFull
	ErrIllegalTftpOp
	ErrUnknownTransferId
	ErrFileAlreadyExists
	ErrNoSuchUser
)

const MaxBlocks = 65535
const MaxPayloadSize = 512
const Datagramsize = 516
