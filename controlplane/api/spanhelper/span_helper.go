package spanhelper

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

// LogFromSpan - return a logger that has a TraceHook to also log messages to the span
func LogFromSpan(span opentracing.Span) logrus.FieldLogger {
	if span != nil {
		logger := logrus.New().WithField("span", span)
		logger.Logger.AddHook(NewTraceHook(span))
		return logger
	}
	return logrus.New()
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

// SpanHelper - wrap span if specified to simplify workflow
type SpanHelper interface {
	Finish()
	Context() context.Context
	Logger() logrus.FieldLogger
	LogObject(attribute string, value interface{})
	LogValue(attribute string, value interface{})
	LogError(err error)
	Span() opentracing.Span
}

type spanHelper struct {
	operation string
	span      opentracing.Span
	ctx       context.Context
	logger    logrus.FieldLogger
}

func (s *spanHelper) Span() opentracing.Span {
	return s.span
}

func (s *spanHelper) LogError(err error) {
	if s.span != nil && err != nil {
		s.span.LogFields(log.Error(err))
		s.Logger().Error(err)
	}
}

func (s *spanHelper) LogObject(attribute string, value interface{}) {
	cc, err := json.Marshal(value)
	msg := value
	if err == nil {
		msg = string(cc)
	}

	if s.span != nil {
		s.span.LogFields(log.Object(attribute, msg))
	} else {
		s.Logger().Infof("%s %v", attribute, msg)
	}
}
func (s *spanHelper) LogValue(attribute string, value interface{}) {
	if s.span != nil {
		s.span.LogFields(log.Object(attribute, value))
	}
}

func (s *spanHelper) Finish() {
	if s.span != nil {
		s.span.Finish()
	}
}

func (s *spanHelper) Logger() logrus.FieldLogger {
	if s.logger == nil {
		s.logger = LogFromSpan(s.span)
		if s.operation != "" {
			s.logger = s.logger.WithField("operation", s.operation)
		}
	}
	return s.logger
}

func NewSpanHelper(ctx context.Context, span opentracing.Span, operation string) SpanHelper {
	return &spanHelper{
		ctx:       ctx,
		span:      span,
		operation: operation,
	}
}

func (s *spanHelper) startSpan(operation string) SpanHelper {
	if s.ctx != nil && tools.IsOpentracingEnabled() {
		newSpan, newCtx := opentracing.StartSpanFromContext(s.ctx, operation)
		return NewSpanHelper(newCtx, newSpan, operation)
	}
	return NewSpanHelper(context.Background(), nil, operation)
}

func (s *spanHelper) Context() context.Context {
	return s.ctx
}

// FromContext - return span helper from context and if opentracing is enabled start new span
func FromContext(ctx context.Context, operation string) SpanHelper {
	if tools.IsOpentracingEnabled() {
		newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
		return NewSpanHelper(newCtx, newSpan, operation)
	}
	// return just context
	return NewSpanHelper(ctx, nil, operation)
}

// GetSpanHelper - construct a span helper object from current context span
func GetSpanHelper(ctx context.Context) SpanHelper {
	if tools.IsOpentracingEnabled() {
		span := opentracing.SpanFromContext(ctx)
		return NewSpanHelper(ctx, span, "")
	}
	// return just context
	return &spanHelper{
		span: nil,
		ctx:  ctx,
	}
}

//CopySpan - construct span helper object with ctx and copy span from spanContext
// Will start new operation on span
func CopySpan(ctx context.Context, spanContext SpanHelper, operation string) SpanHelper {
	return WithSpan(ctx, spanContext.Span(), operation)
}

// WithSpan - construct span helper object with ctx and copy spanid from span
// Will start new operation on span
func WithSpan(ctx context.Context, span opentracing.Span, operation string) SpanHelper {
	if tools.IsOpentracingEnabled() && span != nil {
		ctx = opentracing.ContextWithSpan(ctx, span)
		newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
		return NewSpanHelper(newCtx, newSpan, operation)
	}
	return NewSpanHelper(ctx, nil, operation)
}
