package utils

import (
	"fmt"
	"os"
)

func UserHomeDirPath() string {
	p, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("error while creating getting user home dir: %w", err))
	}

	tftpBaseDir := fmt.Sprintf("%s/tftp", p)

	if _, err := os.Stat(tftpBaseDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(tftpBaseDir, 0o750); err != nil {
				panic(fmt.Errorf("error while creating tftp base dir: %w", err))
			}
		} else {
			panic(fmt.Errorf("error cheking if file exists: %w", err))
		}
	}

	return tftpBaseDir
}
