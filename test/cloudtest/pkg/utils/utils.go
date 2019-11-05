package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
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
	loghook *logrusHook
}

// NewLogKeeper - creates new instance of logrus log collector
func NewLogKeeper() *logKeeper {
	return &logKeeper{
		loghook: newGlobal(),
	}
}

// GetMessages - return a string representation of all messages.
func (lk *logKeeper) GetMessages() []string {
	messages := []string{}
	for _, entry := range lk.loghook.AllEntries() {
		messages = append(messages, entry.Message+"\n")
	}
	return messages
}

// CheckMessagesOrder - checks that messages are present in log in given oreder.
func (lk *logKeeper) CheckMessagesOrder(messages []string) bool {
	if len(messages) == 0 {
		return false
	}
	ind := 0
	msgs := []string{}
	for _, entry := range lk.loghook.AllEntries() {
		msgs = append(msgs, fmt.Sprintf("\"%s\",\n", entry.Message))
		if strings.Contains(entry.Message, messages[ind]) {
			ind++
			if ind == len(messages) {
				return true
			}
		}
	}
	logrus.Infof("Not matched: %v", msgs)
	return false
}

func (lk *logKeeper) Stop() {
	lk.loghook.Stop()
}

// logrusHook is a hook designed for dealing with logs in test scenarios.
type logrusHook struct {
	// Entries is an array of all entries that have been received by this hook.
	// For safe access, use the AllEntries() method, rather than reading this
	// value directly.
	Entries []logrus.Entry
	mu      sync.RWMutex
	stopped bool

	logrus.Hook
}

// NewGlobal installs a test hook for the global logger.
func newGlobal() *logrusHook {
	hook := new(logrusHook)
	hook.stopped = false
	logrus.AddHook(hook)

	return hook
}

func (t *logrusHook) Fire(e *logrus.Entry) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		// If stopped, we do not need to capture any more events.
		return nil
	}
	t.Entries = append(t.Entries, *e)
	return nil
}

func (t *logrusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// LastEntry returns the last entry that was logged or nil.
func (t *logrusHook) LastEntry() *logrus.Entry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	i := len(t.Entries) - 1
	if i < 0 {
		return nil
	}
	return &t.Entries[i]
}

// AllEntries returns all entries that were logged.
func (t *logrusHook) AllEntries() []*logrus.Entry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// Make a copy so the returned value won't race with future log requests
	entries := make([]*logrus.Entry, len(t.Entries))
	for i := 0; i < len(t.Entries); i++ {
		// Make a copy, for safety
		entries[i] = &t.Entries[i]
	}
	return entries
}

// Reset removes all Entries from this test hook.
func (t *logrusHook) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Entries = make([]logrus.Entry, 0)
}

func (t *logrusHook) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
}
