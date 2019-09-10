package dataplane

import (
	"context"

	"github.com/sirupsen/logrus"
)

const loggerKey = "loggerKey"

func Logger(ctx context.Context) logrus.FieldLogger {
	if logger, ok := ctx.Value(loggerKey).(logrus.FieldLogger); ok {
		return logger
	}
	return logrus.New()
}
