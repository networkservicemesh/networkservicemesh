package fanout

import "errors"

var (
	errNoHealthy    = errors.New("no healthy clients")
	errCachedClosed = errors.New("cached connection was closed by peer")
)
