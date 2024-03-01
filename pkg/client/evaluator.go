package client

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type Evaluator struct {
	l      *zap.SugaredLogger
	client Connector
	line   string
}

func NewEvaluator(l *zap.SugaredLogger, client Connector) *Evaluator {
	return &Evaluator{
		l:      l,
		client: client,
	}
}

func (e *Evaluator) evaluate() (bool, error) {
	e.line = strings.TrimSuffix(e.line, "\n")

	cmd := strings.Split(e.line, " ")

	switch cmd[0] {
	case "connect":
		if len(cmd) > 3 {
			return false, errors.New(fmt.Sprintf("connect command received too many arguments: %s", e.line))
		}
		e.l.Debug(cmd)
		return false, e.client.Connect(fmt.Sprintf("%s:%s", cmd[1], cmd[2]))
	case "trace":
		if len(cmd) > 2 {
			return false, errors.New(fmt.Sprintf("trace command received too many arguments: %s", e.line))
		}

		e.client.SetTrace()
	case "get":
		if len(cmd) > 2 {
			return false, errors.New(fmt.Sprintf("get command received too many arguments: %s", e.line))
		}

		return false, e.client.Get(cmd[1])
	case "put":
		if len(cmd) > 2 {
			return false, errors.New(fmt.Sprintf("put command received too many arguments: %s", e.line))
		}

		return false, e.client.Put(cmd[1])
	case "quit":
		if len(cmd) > 1 {
			return false, errors.New(fmt.Sprintf("quit command received too many arguments: %s", e.line))
		}

		return true, nil
	default:
		return false, errors.New(fmt.Sprintf("unkown command; %s", e.line))
	}

	return false, nil
}
