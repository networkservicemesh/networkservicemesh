// Copyright (c) 2019 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
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
}

type spanHelper struct {
	span   opentracing.Span
	ctx    context.Context
	logger logrus.FieldLogger
}

func (s *spanHelper) LogError(err error) {
	if s.span != nil && err != nil {
		s.span.LogFields(log.Error(err))
		s.Logger().Error(err)
	}
}

func (s *spanHelper) LogObject(attribute string, value interface{}) {
	if s.span != nil {
		cc, err := json.Marshal(value)
		if err == nil {
			s.span.LogFields(log.Object(attribute, string(cc)))
		} else {
			s.span.LogFields(log.Object(attribute, value))
		}
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
	}
	return s.logger
}

func (s *spanHelper) startSpan(operation string) SpanHelper {
	if s.span != nil && tools.IsOpentracingEnabled() {
		newSpan, newCtx := opentracing.StartSpanFromContext(s.ctx, operation)
		return &spanHelper{
			ctx:  newCtx,
			span: newSpan,
		}
	}
	return &spanHelper{
		ctx:  context.Background(),
		span: nil,
	}
}

func (s *spanHelper) Context() context.Context {
	if s.span == nil {
		return context.Background()
	}
	return s.ctx
}

// SpanHelperFromContext - return span helper from context and if opentracing is enabled start new span
func SpanHelperFromContext(ctx context.Context, operation string) SpanHelper {
	if tools.IsOpentracingEnabled() {
		newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
		return &spanHelper{
			span: newSpan,
			ctx:  newCtx,
		}
	}
	// return just context
	return &spanHelper{
		span: nil,
		ctx:  ctx,
	}
}

// GetSpanHelper - construct a span helper object from current context span
func GetSpanHelper(ctx context.Context) SpanHelper {
	if tools.IsOpentracingEnabled() {
		return &spanHelper{
			span: opentracing.SpanFromContext(ctx),
			ctx:  ctx,
		}
	}
	// return just context
	return &spanHelper{
		span: nil,
		ctx:  ctx,
	}
}

//SpanHelperFromContextCopySpan - construct span helper object with ctx and copy span from spanContext
// Will start new operation on span
func SpanHelperFromContextCopySpan(ctx context.Context, spanContext SpanHelper, operation string) SpanHelper {
	if tools.IsOpentracingEnabled() {
		span := opentracing.SpanFromContext(spanContext.Context())
		if span != nil {
			ctx = opentracing.ContextWithSpan(ctx, span)
			newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
			return &spanHelper{
				span: newSpan,
				ctx:  newCtx,
			}
		}
	}
	return &spanHelper{
		span: nil,
		ctx:  ctx,
	}
}

// SpanHelperFromConnection - construct new span helper with span from context is pressent or span from connection object
func SpanHelperFromConnection(ctx context.Context, clientConnection *model.ClientConnection, operation string) SpanHelper {
	result := &spanHelper{
		span: opentracing.SpanFromContext(ctx),
		ctx:  ctx,
	}
	if !tools.IsOpentracingEnabled() {
		return result
	}
	if result.span != nil {
		// Context already had span, so let's just start operation.
		return result.startSpan(operation)
	}
	if clientConnection != nil && clientConnection.Span != nil {
		ctx = opentracing.ContextWithSpan(ctx, clientConnection.Span)
		span, newCtx := opentracing.StartSpanFromContext(ctx, operation)

		result.span = span
		result.ctx = newCtx
		return result
	}

	// No connection object and no span in context, lets start new
	newSpan, newCtx := opentracing.StartSpanFromContext(ctx, operation)
	return &spanHelper{
		span: newSpan,
		ctx:  newCtx,
	}
}
