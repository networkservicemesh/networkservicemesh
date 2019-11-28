// Copyright (c) 2019 Cisco Systems, Inc and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
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

// Package utils - Utils for cloud testing tool
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

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
func (lk *logKeeper) CheckMessagesOrder(t *testing.T, messages []string) bool {
	if len(messages) == 0 {
		return false
	}
	ind := 0
	msgs := []string{}
	matched := []string{}
	lastMatched := 0
	for _, entry := range lk.loghook.AllEntries() {
		msgs = append(msgs, fmt.Sprintf("\"%s\",\n", entry.Message))
		if ind < len(messages) && strings.Contains(entry.Message, messages[ind]) {
			matched = append(matched, fmt.Sprintf("'%s' contains in '%s'", messages[ind], entry.Message))
			ind++
			lastMatched = len(msgs) - 1
			if ind == len(messages) {
				return true
			}
		}
	}

	logrus.Infof("Matched %v", matched)
	res := ""
	for _, um := range messages[ind:] {
		res += "\n" + um
	}
	res += "\nTail:\n"
	for _, um := range msgs[lastMatched:] {
		res += um
	}
	t.Fatalf("Unmatched: %v", res)
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

// MatchRetestPattern - check if retest pattern is matched in passed string
func MatchRetestPattern(patterns []string, line string) bool {
	for _, pp := range patterns {
		// Check for contains
		if strings.Contains(line, pp) {
			return true
		}
		// Check using regexp
		if matched, _ := regexp.MatchString(pp, line); matched {
			return true
		}
	}
	return false
}
