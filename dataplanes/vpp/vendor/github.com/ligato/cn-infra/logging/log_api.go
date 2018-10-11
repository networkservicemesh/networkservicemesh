// Copyright (c) 2017 Cisco and/or its affiliates.
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

package logging

import "fmt"

// Fields is a type accepted by WithFields method. It can be used to instantiate map using shorter notation.
type Fields map[string]interface{}

// LogLevel represents severity of log record
type LogLevel uint32

const (
	// PanicLevel - highest level of severity. Logs and then calls panic with the message passed in.
	PanicLevel LogLevel = iota
	// FatalLevel - logs and then calls `os.Exit(1)`.
	FatalLevel
	// ErrorLevel - used for errors that should definitely be noted.
	ErrorLevel
	// WarnLevel - non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel - general operational entries about what's going on inside the application.
	InfoLevel
	// DebugLevel - enabled for debugging, very verbose logging.
	DebugLevel
)

// String converts the LogLevel to a string. E.g. PanicLevel becomes "panic".
func (level LogLevel) String() string {
	switch level {
	case PanicLevel:
		return "panic"
	case FatalLevel:
		return "fatal"
	case ErrorLevel:
		return "error"
	case WarnLevel:
		return "warning"
	case InfoLevel:
		return "info"
	case DebugLevel:
		return "debug"
	}

	return fmt.Sprintf("unknown(%d)", level)
}

// Logger provides logging capabilities
type Logger interface {
	// GetName return the logger name
	GetName() string
	// SetLevel modifies the LogLevel
	SetLevel(level LogLevel)
	// GetLevel returns currently set logLevel
	GetLevel() LogLevel
	// WithField creates one structured field
	WithField(key string, value interface{}) LogWithLevel
	// WithFields creates multiple structured fields
	WithFields(fields map[string]interface{}) LogWithLevel

	LogWithLevel
}

// LogWithLevel allows to log with different log levels
type LogWithLevel interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatalln(args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})

	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// LogFactory is API for the plugins that want to create their own loggers.
type LogFactory interface {
	NewLogger(name string) Logger
}

// PluginLogger is intended for:
// 1. small plugins (that just need one logger; name corresponds to plugin name)
// 2. large plugins that need multiple loggers (all loggers share same name prefix)
type PluginLogger interface {
	// Plugin has by default possibility to log
	// Logger name is initialized with plugin name
	Logger

	// LogFactory can be optionally used by large plugins
	// to create child loggers (their names are prefixed by plugin logger name)
	LogFactory
}

// Registry groups multiple Logger instances and allows to mange their log levels.
type Registry interface {
	// LogFactory allow to create new loggers
	LogFactory
	// List Loggers returns a map (loggerName => log level)
	ListLoggers() map[string]string
	// SetLevel modifies log level of selected logger in the registry
	SetLevel(logger, level string) error
	// GetLevel returns the currently set log level of the logger from registry
	GetLevel(logger string) (string, error)
	// Lookup returns a logger instance identified by name from registry
	Lookup(loggerName string) (logger Logger, found bool)
	// ClearRegistry removes all loggers except the default one from registry
	ClearRegistry()
}

// ForPlugin is used to initialize plugin logger by name
// and optionally created children (their name prefixed by plugin logger name)
//
// Example usage:
//
//    flavor.ETCD.Logger =
// 			ForPlugin(PluginNameOfFlavor(&flavor.ETCD, flavor), flavor.Logrus)
//
func ForPlugin(name string, factory LogFactory) PluginLogger {
	return &pluginLogger{
		Logger:     factory.NewLogger(name),
		LogFactory: &prefixedLogFactory{name, factory},
	}
}

func (factory *prefixedLogFactory) NewLogger(name string) Logger {
	return factory.delegate.NewLogger(factory.prefix + name)
}

type prefixedLogFactory struct {
	prefix   string
	delegate LogFactory
}

type pluginLogger struct {
	Logger
	LogFactory
}
