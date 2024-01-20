package utils

import "errors"

var (
	ErrStartingServer         = errors.New("error while starting the udp server")
	ErrWrongOpCode            = errors.New("invalid operation code")
	ErrDataPayloadTooBig      = errors.New("payload exceeds 512 bytes")
	ErrDataMarshall           = errors.New("error while marshall data packet")
	ErrDataPacketCanNotBeSent = errors.New("data packet can not be sent")
)
