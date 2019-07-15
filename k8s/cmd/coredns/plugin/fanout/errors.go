package fanout

import "errors"

var (
	errNoHealthy    = errors.New("no healthy agents")
	errNoForward    = errors.New("no agent defined")
	errCachedClosed = errors.New("cached connection was closed by peer")
)
