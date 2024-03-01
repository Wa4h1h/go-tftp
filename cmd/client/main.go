package main

import (
	"context"
	"sync"

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

	c.SetTrace(true)

	wg := sync.WaitGroup{}

	wg.Add(1)

	go func() {
		if err := c.Put(context.Background(), "Kubernetes.pdf"); err != nil {
			l.Error(err)
		}

		wg.Done()
	}()

	wg.Wait()
}
