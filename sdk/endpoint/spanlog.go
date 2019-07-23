package endpoint

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
)

func LogFromSpan(span opentracing.Span) logrus.FieldLogger {
	logger := logrus.New().WithField("span", span)
	logger.Logger.AddHook(NewTraceHook(span))
	return logger
}

type traceHook struct {
	index int
	span  opentracing.Span
}

func NewTraceHook(span opentracing.Span) logrus.Hook {
	return &traceHook{
		span: span,
	}
}

func (h *traceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *traceHook) Fire(entry *logrus.Entry) error {
	msg, err := entry.String()
	if err != nil {
		return err
	}
	h.span.LogFields(log.String(fmt.Sprintf("log[%d]", h.index), msg))
	h.index++
	return nil
}
