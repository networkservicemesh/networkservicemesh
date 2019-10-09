package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// Contains - check if array contains element.
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// NewRandomStr - generates random string of desired length, size should be multiple of two for best result.
func NewRandomStr(size int) string {
	value := make([]byte, size/2)
	_, err := rand.Read(value)
	if err != nil {
		logrus.Errorf("error during random string generation %v", err)
		return ""
	}
	return hex.EncodeToString(value)
}

type logKeeper struct {
	loghook *test.Hook
}

// NewLogKeeper - creates new instance of logrus log collector
func NewLogKeeper() *logKeeper {
	return &logKeeper{
		loghook: test.NewGlobal(),
	}
}

// CheckMessagesOrder - checks that messages are present in log in given oreder.
func (lk *logKeeper) CheckMessagesOrder(messages []string) bool {
	if len(messages) == 0 {
		return false
	}
	ind := 0
	for _, entry := range lk.loghook.AllEntries() {
		if strings.Contains(entry.Message, messages[ind]) {
			ind++
			if ind == len(messages) {
				return true
			}
		}
	}
	return false
}
