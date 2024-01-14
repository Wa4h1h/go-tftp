package utils

import "errors"

var (
	ErrStartingServer        = errors.New("error while starting the udp server")
	ErrReadingFromConnection = errors.New("error while reading from connection")
)
