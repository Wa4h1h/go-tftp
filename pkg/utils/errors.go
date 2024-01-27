package utils

import "errors"

var (
	ErrStartingServer        = errors.New("error: starting the udp server")
	ErrWrongOpCode           = errors.New("error: invalid operation code")
	ErrDataPayloadTooBig     = errors.New("error: payload exceeds 512 bytes")
	ErrPacketMarshall        = errors.New("error: can marshall packet")
	ErrPacketCanNotBeSent    = errors.New("error: packet can not be sent")
	ErrCanNotSetWriteTimeout = errors.New("error: can not set write timeout")
	ErrCanNotSetReadTimeout  = errors.New("error: can not set read timeout")
	ErrCanNotCopySLice       = errors.New("error: can not copy slice")
)
