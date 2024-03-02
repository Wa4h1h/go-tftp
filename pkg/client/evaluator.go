package client

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

var (
	getRegex     = "^get\\s+([\\S\\s]+)$"
	putRegex     = "^put\\s+([\\S\\s]+)$"
	timeoutRegex = "^timeout\\s+(\\d+)$"
	connectRegex = "^connect\\s+([\\S\\s]+)\\s+([\\S\\s]+)$"
	traceRegex   = "^trace$"
	quitRegex    = "^quit$"
	helpRegex    = "^help$"
)

type Evaluator struct {
	l             *zap.SugaredLogger
	client        Connector
	line          string
	regexPatterns map[string]*regexp.Regexp
}

func NewEvaluator(l *zap.SugaredLogger, client Connector) *Evaluator {
	e := &Evaluator{
		l:      l,
		client: client,
	}

	e.regexPatterns = make(map[string]*regexp.Regexp)

	e.regexPatterns["get"] = regexp.MustCompile(getRegex)
	e.regexPatterns["put"] = regexp.MustCompile(putRegex)
	e.regexPatterns["timeout"] = regexp.MustCompile(timeoutRegex)
	e.regexPatterns["connect"] = regexp.MustCompile(connectRegex)
	e.regexPatterns["trace"] = regexp.MustCompile(traceRegex)
	e.regexPatterns["quit"] = regexp.MustCompile(quitRegex)
	e.regexPatterns["help"] = regexp.MustCompile(helpRegex)

	return e
}

func (e *Evaluator) evaluate() (bool, error) {
	e.line = strings.TrimSuffix(e.line, "\n")

	if matches := e.regexPatterns["get"].FindStringSubmatch(e.line); len(matches) == 2 {
		return false, e.client.Get(matches[1])
	}

	if matches := e.regexPatterns["put"].FindStringSubmatch(e.line); len(matches) == 2 {
		return false, e.client.Put(matches[1])
	}

	if matches := e.regexPatterns["timeout"].FindStringSubmatch(e.line); len(matches) == 2 {
		n, err := strconv.ParseUint(matches[1], 10, 32)
		if err != nil {
			return false, fmt.Errorf("timeout value can not be parsed: %w", err)
		}

		e.client.SetTimeout(uint(n))

		return false, nil
	}

	if matches := e.regexPatterns["connect"].FindStringSubmatch(e.line); len(matches) == 3 {
		return false, e.client.Connect(fmt.Sprintf("%s:%s", matches[1], matches[2]))
	}

	if matches := e.regexPatterns["trace"].FindStringSubmatch(e.line); len(matches) == 1 {
		e.client.SetTrace()

		return false, nil
	}

	if matches := e.regexPatterns["help"].FindStringSubmatch(e.line); len(matches) == 1 {
		fmt.Println(`Commands:
	connect <host> <port>
	get <file>
	put <file>
	timeout <integer>
	trace
	quit`)
		return false, nil
	}

	if matches := e.regexPatterns["quit"].FindStringSubmatch(e.line); len(matches) == 1 {
		return true, nil
	}

	return false, errors.New(fmt.Sprintf("unknow command  arguments: %s", e.line))
}
