package filename

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

type Hook struct {
	Field        string
	Skip         int
	levels       []logrus.Level
	SkipPrefixes []string
	Formatter    func(file, function string, line int) string
}

func (hook *Hook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *Hook) Fire(entry *logrus.Entry) error {
	entry.Data[hook.Field] = hook.Formatter(hook.findCaller())
	return nil
}

func (hook *Hook) findCaller() (string, string, int) {
	var (
		pc       uintptr
		file     string
		function string
		line     int
	)
	for i := 0; i < 10; i++ {
		pc, file, line = getCaller(hook.Skip + i)
		if !hook.skipFile(file) {
			break
		}
	}
	if pc != 0 {
		frames := runtime.CallersFrames([]uintptr{pc})
		frame, _ := frames.Next()
		function = frame.Function
	}

	return file, function, line
}

func (hook *Hook) skipFile(file string) bool {
	for i := range hook.SkipPrefixes {
		if strings.HasPrefix(file, hook.SkipPrefixes[i]) {
			return true
		}
	}

	return false
}

func NewHook(levels ...logrus.Level) *Hook {
	hook := Hook{
		Field:        "source",
		Skip:         5,
		levels:       levels,
		SkipPrefixes: []string{"logrus/", "logrus@"},
		Formatter: func(file, function string, line int) string {
			return fmt.Sprintf("%s:%d", file, line)
		},
	}
	if len(hook.levels) == 0 {
		hook.levels = logrus.AllLevels
	}

	return &hook
}

func getCaller(skip int) (uintptr, string, int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return 0, "", 0
	}

	n := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			n++
			if n >= 2 {
				file = file[i+1:]
				break
			}
		}
	}

	return pc, file, line
}
