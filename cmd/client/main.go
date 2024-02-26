package main

import (
	"context"

	"github.com/Wa4h1h/go-tftp/pkg/client"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
)

var (
	logLevel = utils.GetEnv[string]("TFTP_LOG_LEVEL", "debug", false)
	numTries = utils.GetEnv[uint]("TFTP_NUM_TRIES", "5", false)
)

func main() {
	l := utils.NewLogger(logLevel).Sugar()
	c := client.NewClient(l, numTries)

	if err := c.Connect("127.0.0.1:69"); err != nil {
		l.Error(err)
	}

	defer func(client client.Connector) {
		if err := client.Close(); err != nil {
			l.Error(err.Error())
		}
	}(c)

	if err := c.Get(context.Background(), "main-concepts.pdf"); err != nil {
		l.Error(err)
	}

	if err := c.Get(context.Background(), "Kubernetes.pdf"); err != nil {
		l.Error(err)
	}
}
