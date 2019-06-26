package runners

import (
	"bufio"
	"context"
)

// TestRunner - describes a way to execute tests.
type TestRunner interface {
	// Run - run tests with timeout context, environment
	Run(timeoutCtx context.Context, env [] string, writer *bufio.Writer) error
	// GetCmdLine - return created command line, if applicable.
	GetCmdLine() string
}
