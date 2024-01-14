package main

import (
	"fmt"
	"github.com/WadhahJemai/go-tftp/internal/servers/tftp"
	"github.com/WadhahJemai/go-tftp/internal/utils"
	"os"
	"os/signal"
	"syscall"
)

var (
	tftpPort     = utils.GetEnv[string]("TFTP_PORT", "69", false)
	logLevel     = utils.GetEnv[string]("LOG_LEVEL", "debug", false)
	readTimeout  = utils.GetEnv[uint]("READ_TIMEOUT", "5", false)
	writeTimeout = utils.GetEnv[uint]("WRITE_TIMEOUT", "5", false)
	numTries     = utils.GetEnv[uint]("NUM_TRIES", "5", false)
)

func main() {
	l := utils.NewLogger(logLevel)
	s := tftp.NewServer(l, tftpPort, readTimeout, writeTimeout)

	go func(se *tftp.Server) {
		if err := se.ListenAndServe(); err != nil {
			l.Error(err.Error())
		}
	}(s)

	l.Info(fmt.Sprintf("listening on port %s", tftpPort))

	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}

		l.Info(fmt.Sprintf("closed connection on port %s", tftpPort))
	}()

	// listen shutdown signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
}
