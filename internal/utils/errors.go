package utils

import "errors"

var (
	ErrStartingServer         = errors.New("error: starting the udp server")
	ErrWrongOpCode            = errors.New("error: invalid operation code")
	ErrDataPayloadTooBig      = errors.New("error: payload exceeds 512 bytes")
	ErrDataMarshall           = errors.New("error: can marshall data packet")
	ErrDataPacketCanNotBeSent = errors.New("error: data packet can not be sent")
	ErrCanNotSetWriteTimeout  = errors.New("error: can not set write timeout")
	ErrCanNotSetReadTimeout   = errors.New("error: can not set read timeout")
)
