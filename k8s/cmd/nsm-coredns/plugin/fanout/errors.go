package fanout

import "github.com/pkg/errors"

var (
	errContextDone  = errors.New("context is done")
	errNoHealthy    = errors.New("no healthy clients")
	errCachedClosed = errors.New("cached connection was closed by peer")
)
