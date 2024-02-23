package main

import (
	"github.com/Wa4h1h/go-tftp/pkg/client"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
)

var (
	logLevel = utils.GetEnv[string]("TFTP_LOG_LEVEL", "debug", false)
)

func main() {
	l := utils.NewLogger(logLevel).Sugar()
	c := client.NewClient(l)

	c.Connect("localhost:69")
}
