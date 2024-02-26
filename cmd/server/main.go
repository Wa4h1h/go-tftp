package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Wa4h1h/go-tftp/pkg/server"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
)

var (
	tftpPort     = utils.GetEnv[string]("TFTP_PORT", "69", false)
	logLevel     = utils.GetEnv[string]("TFTP_LOG_LEVEL", "debug", false)
	readTimeout  = utils.GetEnv[uint]("TFTP_READ_TIMEOUT", "5", false)
	writeTimeout = utils.GetEnv[uint]("TFTP_WRITE_TIMEOUT", "5", false)
	numTries     = utils.GetEnv[uint]("TFTP_NUM_TRIES", "5", false)
	tftpBaseDir  = utils.GetEnv[string]("TFTP_BASE_DIR", utils.UserHomeDirPath(), false)
)

func main() {
	l := utils.NewLogger(logLevel).Sugar()
	s := server.NewServer(l, tftpPort, readTimeout, writeTimeout, int(numTries), tftpBaseDir)

	go func() {
		if err := s.ListenAndServe(); err != nil {
			l.Error(err.Error())
		}
	}()

	l.Infof("listening on port %s", tftpPort)

	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}

		l.Infof("closed connection on port %s", tftpPort)
	}()

	// listen shutdown signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
}
