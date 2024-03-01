package client

import (
	"os"
)

func checkFileExist(file string) bool {
	_, err := os.Stat(file)

	return err == nil
}
