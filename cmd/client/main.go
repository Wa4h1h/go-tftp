package main

import (
	"github.com/Wa4h1h/go-tftp/pkg/client"
	"github.com/Wa4h1h/go-tftp/pkg/utils"
)

var (
	logLevel = utils.GetEnv[string]("TFTP_LOG_LEVEL", "debug", false)
	numTries = utils.GetEnv[uint]("TFTP_NUM_TRIES", "5", false)
)

func main() {
	l := utils.NewLogger(logLevel).Sugar()
	tftp := client.NewClient(l, numTries)
	c := client.NewCli(l, tftp)

	c.Read()
	/*getRegex := "^put\\s+(\\S+)$"
	r := regexp.MustCompile(getRegex)
	fmt.Println(r.FindStringSubmatch("put Kubernetes.pdf"))*/
}
