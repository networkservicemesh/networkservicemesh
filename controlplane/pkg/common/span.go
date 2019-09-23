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
	"fmt"

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
