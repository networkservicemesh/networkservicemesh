package runners

import (
	"bufio"
	"context"
)

type TestRunner interface {
	Run(timeoutCtx context.Context, env [] string, fileName string, writer *bufio.Writer) error
	GetCmdLine() string
}
