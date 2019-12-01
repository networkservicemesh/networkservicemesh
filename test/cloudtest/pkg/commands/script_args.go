package commands

import (
	"bufio"
	"time"
)

const runScriptTimeout = time.Minute * 3

type runScriptArgs struct {
	Name, ClusterTaskId, Script string
	Env                         []string
	Out                         *bufio.Writer
}
