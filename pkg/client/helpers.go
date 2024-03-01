package client

import (
	"os"
)

func checkFileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}

	return true
}
