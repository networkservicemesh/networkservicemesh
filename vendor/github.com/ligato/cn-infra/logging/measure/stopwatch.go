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

package measure

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/ligato/cn-infra/logging"
)

// StopWatchEntry provides method to log measured time entries
type StopWatchEntry interface {
	LogTimeEntry(d time.Duration)
}

// Stopwatch keeps all time measurement results
type Stopwatch struct {
	// name of the entity/plugin
	name string
	// logger used while printing
	logger logging.Logger
	// map where measurements are stored. Map is in format [string]TimeLog (string is a name related to the measured time(s)
	// which are stored in timelog), for every binapi/netlink api there is
	// a set of times this binapi/netlink was called
	timeTable sync.Map
}

// NewStopwatch creates a new stopwatch object with empty time map
func NewStopwatch(name string, log logging.Logger) *Stopwatch {
	return &Stopwatch{
		name:      name,
		logger:    log,
		timeTable: sync.Map{},
	}
}

// TimeLog is a wrapper for the measured data for specific name
type TimeLog struct {
	entries []time.Duration
}

// GetTimeLog returns a pointer to the TimeLog object related to the provided name (derived from the <n> parameter).
// If stopwatch is not used, returns nil
func GetTimeLog(n interface{}, s *Stopwatch) *TimeLog {
	// return nil if does not exist
	if s == nil {
		return nil
	}
	return s.timeLog(n)
}

// TimeLog returns a pointer to the TimeLog object related to the provided name (derived from the <n> parameter).
// If stopwatch instance is nil, returns nil
func (st *Stopwatch) TimeLog(n interface{}) *TimeLog {
	if st == nil {
		return nil
	}
	return st.timeLog(n)
}

// looks over stopwatch timeTable map in order to find a TimeLog object for provided name. If the object does not exist,
// it is created anew, stored in the map and returned
func (st *Stopwatch) timeLog(n interface{}) *TimeLog {
	// derive name
	var name string
	switch nType := n.(type) {
	case string:
		name = nType
	default:
		name = reflect.TypeOf(n).String()
	}
	// create and initialize new TimeLog in case it does not exist
	timer := &TimeLog{}
	timer.entries = make([]time.Duration, 0)
	// if there is no TimeLog under the name, store the created one. Otherwise, existing timer is returned
	existingTimer, loaded := st.timeTable.LoadOrStore(name, timer)
	if loaded {
		// cast to object which can be returned
		existing, ok := existingTimer.(*TimeLog)
		if !ok {
			panic(fmt.Errorf("cannot cast timeTable map value to duration"))
		}
		return existing
	}
	return timer
}

// LogTimeEntry stores time entry to the TimeLog (the time log itself is stored in the stopwatch sync.Map)
func (t *TimeLog) LogTimeEntry(d time.Duration) {
	if t != nil && t.entries != nil {
		t.entries = append(t.entries, d)
	}
}

// LogTimeEntryFor sets start time and returns func which logs the time entry,
// usable as single-line defer call: `defer s.LogTimeEntryFor("xyz")()`
func (st *Stopwatch) LogTimeEntryFor(n interface{}) func() {
	if st == nil {
		return func() {}
	}
	l := st.timeLog(n)
	start := time.Now()
	return func() {
		l.LogTimeEntry(time.Since(start))
	}
}

// PrintLog all entries from TimeLog and reset it
func (st *Stopwatch) PrintLog() {
	isMapEmpty := true
	var wasErr error
	// Calculate stTotal time
	var stTotal time.Duration
	st.timeTable.Range(func(k, v interface{}) bool {
		// Remember that the map contained entries
		isMapEmpty = false
		name, ok := k.(string)
		if !ok {
			wasErr = fmt.Errorf("cannot cast timeTable map key to string")
			// stops the iteration
			return false
		}
		value, ok := v.(*TimeLog)
		if !ok {
			wasErr = fmt.Errorf("cannot cast timeTable map value to duration")
			// stops the iteration
			return false
		}
		// Calculate average value of entry list
		nameTotal, average := st.calculateAverage(value.entries)
		// Add to total
		stTotal += nameTotal
		st.logger.WithFields(logging.Fields{"conf": st.name, "wasCalled": len(value.entries),
			"durationInNs": average.Nanoseconds()}).Infof("%v call took %v", name, average)

		return true
	})

	// throw panic outside of logger.Range()
	if wasErr != nil {
		panic(wasErr)
	}

	// In case map is entry
	if isMapEmpty {
		st.logger.WithField("conf", st.name).Infof("stopwatch has no entries")
	}
	// Log overall time
	st.logger.WithFields(logging.Fields{"conf": st.name, "durationInNs": stTotal.Nanoseconds()}).Infof("partial resync time is %v", stTotal)

	// clear map after use
	st.timeTable = sync.Map{}
}

// calculates average duration of binary api + total duration of that binary api (if called more than once)
func (st *Stopwatch) calculateAverage(durations []time.Duration) (total time.Duration, average time.Duration) {
	if len(durations) == 0 {
		return 0, 0
	}
	for _, duration := range durations {
		total += duration
	}

	avgVal := total.Nanoseconds() / int64(len(durations))

	return total, time.Duration(avgVal)
}
