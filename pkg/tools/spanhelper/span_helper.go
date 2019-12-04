package spanhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
)

type spanHelperKeyType string

const (
	spanHelperTraceDepth spanHelperKeyType = "spanHelperTraceDepth"
	maxStringLength                        = 1000
	dotCount                               = 3
)

func withTraceDepth(parent context.Context, value int) context.Context {
	return context.WithValue(parent, spanHelperTraceDepth, value)
}

func traceDepth(ctx context.Context) int {
	if rv, ok := ctx.Value(spanHelperTraceDepth).(int); ok {
		return rv
	}
	return 0
}

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
		debug := limitString(string(debug.Stack()))
		msg := limitString(fmt.Sprintf("%+v", err))
		otgrpc.SetSpanTags(s.span, err, false)
		s.span.LogFields(log.String("event", "error"), log.String("message", msg), log.String("stacktrace", debug))
		logrus.Errorf(">><<%s %s=%v span=%v", strings.Repeat("--", traceDepth(s.ctx)), "error", fmt.Sprintf("%+v", err), s.span)
	}
}

func (s *spanHelper) LogObject(attribute string, value interface{}) {
	cc, err := json.Marshal(value)
	msg := ""
	if err == nil {
		msg = string(cc)
	} else {
		msg = fmt.Sprint(msg)
	}
	msg = limitString(msg)
	if s.span != nil {
		s.span.LogFields(log.Object(attribute, msg))
	}
	logrus.Infof(">><<%s %s=%v span=%v", strings.Repeat("--", traceDepth(s.ctx)), attribute, msg, s.span)
}

func (s *spanHelper) LogValue(attribute string, value interface{}) {
	if s.span != nil {
		msg := limitString(fmt.Sprint(value))
		s.span.LogFields(log.Object(attribute, msg))
	}
	logrus.Infof(">><<%s %s=%v span=%v", strings.Repeat("--", traceDepth(s.ctx)), attribute, value, s.span)
}

func (s *spanHelper) Finish() {
	if s.span != nil {
		s.span.Finish()
		s.span = nil
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

// NewSpanHelper - constructs a span helper from context/snap and opertaion name.
func NewSpanHelper(ctx context.Context, span opentracing.Span, operation string) SpanHelper {
	return &spanHelper{
		ctx:       withTraceDepth(ctx, traceDepth(ctx)+1),
		span:      span,
		operation: operation,
	}
}

func (s *spanHelper) startSpan(operation string) (result SpanHelper) {
	if s.ctx != nil && jaeger.IsOpentracingEnabled() {
		newSpan, newCtx := opentracing.StartSpanFromContext(s.ctx, operation)
		result = NewSpanHelper(newCtx, newSpan, operation)
	} else {
		result = NewSpanHelper(context.Background(), nil, operation)
	}
	result.Logger().Infof("===> %v()", operation)
	return result
}

func (s *spanHelper) Context() context.Context {
	return s.ctx
}

// FromContext - return span helper from context and if opentracing is enabled start new span
func FromContext(ctx context.Context, operation string) (result SpanHelper) {
	if jaeger.IsOpentracingEnabled() {
		newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
		result = NewSpanHelper(newCtx, newSpan, operation)
	} else {
		// return just context
		result = NewSpanHelper(ctx, nil, operation)
	}
	printStart(result, operation)
	return result
}

func printStart(result SpanHelper, operation string) {
	prefix := strings.Repeat("--", traceDepth(result.Context()))
	logrus.Infof("==%s> %v() span:%v", prefix, operation, result.Span())
}

// GetSpanHelper - construct a span helper object from current context span
func GetSpanHelper(ctx context.Context) SpanHelper {
	if jaeger.IsOpentracingEnabled() {
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
func WithSpan(ctx context.Context, span opentracing.Span, operation string) (result SpanHelper) {
	if jaeger.IsOpentracingEnabled() && span != nil {
		ctx = opentracing.ContextWithSpan(ctx, span)
		newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
		result = NewSpanHelper(newCtx, newSpan, operation)
	} else {
		result = NewSpanHelper(ctx, nil, operation)
	}
	printStart(result, operation)
	return result
}

func limitString(s string) string {
	if len(s) > maxStringLength {
		return s[maxStringLength-dotCount:] + strings.Repeat(".", dotCount)
	}
	return s
}
