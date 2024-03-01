package client

import (
	"bufio"
	"fmt"
	"os"

	"go.uber.org/zap"
)

type Cli struct {
	l          *zap.SugaredLogger
	tftpClient Connector
}

func NewCli(l *zap.SugaredLogger, tftpClient Connector) *Cli {
	return &Cli{l: l, tftpClient: tftpClient}
}

func (c *Cli) Read() {
	scanner := bufio.NewScanner(os.Stdin)
	evaluator := NewEvaluator(c.l, c.tftpClient)

	fmt.Print("tftp> ")

	for scanner.Scan() {
		evaluator.line = scanner.Text()

		done, err := evaluator.evaluate()
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}

		if done {
			break
		}

		fmt.Print("tftp> ")
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
