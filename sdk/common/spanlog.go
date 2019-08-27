package common

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
)

// LogFromSpan - return a logger that has a TraceHook to also log messages to the span
func LogFromSpan(span opentracing.Span) logrus.FieldLogger {
	var logger logrus.FieldLogger
	if span != nil {
		l := logrus.New().WithField("span", span)
		logger = l
		l.Logger.AddHook(NewTraceHook(span))
	} else {
		logger = logrus.New()
	}
	return logger
}

type traceHook struct {
	index int
	span  opentracing.Span
}

// NewTraceHook - create a TraceHook for also logging to a span
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
