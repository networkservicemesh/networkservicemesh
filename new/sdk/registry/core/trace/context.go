package trace

import (
	"context"

	"github.com/sirupsen/logrus"
)

type contextKeyType string

const (
	logKey contextKeyType = "Log"
)

// withLog -
//   Provides a FieldLogger in context
func withLog(parent context.Context, log logrus.FieldLogger) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, logKey, log)
}

// Log - return FieldLogger from context
func Log(ctx context.Context) logrus.FieldLogger {
	if rv, ok := ctx.Value(logKey).(logrus.FieldLogger); ok {
		return rv
	}
	return logrus.New()
}
