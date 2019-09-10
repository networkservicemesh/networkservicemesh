package dataplane

import (
	"context"

	"github.com/sirupsen/logrus"
)

type loggerKeyType string

const loggerKey loggerKeyType = "loggerKey"

//Logger returns logger from context
func Logger(ctx context.Context) logrus.FieldLogger {
	if logger, ok := ctx.Value(loggerKey).(logrus.FieldLogger); ok {
		return logger
	}
	return logrus.New()
}
