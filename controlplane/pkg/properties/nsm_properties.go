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

// Package properties - define a set of connection and healing properties
package properties

import (
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// NsmdHealEnabled - environment variable name - disable heal and closes connection
	NsmdHealEnabled = "NSMD_HEAL_ENABLED" // Does healing is enabled or not
	// NsmdHealDSTWaitTimeout - environment variable name - timeout of waiting for networkservice when healing connection
	NsmdHealDSTWaitTimeout = "NSMD_HEAL_DST_TIMEOUTs" // Wait timeout for DST in seconds
	// NsmdHealRetryCount - amount of times healing will retry
	NsmdHealRetryCount = "NSMD_HEAL_RETRY_COUNT"
)

// Properties - holds properties of NSM connection events processing
type Properties struct {
	HealTimeout                    time.Duration
	CloseTimeout                   time.Duration
	HealRequestTimeout             time.Duration
	HealRequestConnectTimeout      time.Duration
	HealRetryCount                 int
	HealRetryDelay                 time.Duration
	HealRequestConnectCheckTimeout time.Duration
	HealForwarderTimeout           time.Duration

	// Total DST heal timeout is 20 seconds.
	HealDSTNSEWaitTimeout time.Duration
	HealDSTNSEWaitTick    time.Duration

	HealEnabled bool
}

// NewNsmProperties creates NsmProperties with defined default values and reading values from environment variables
func NewNsmProperties() *Properties {
	values := &Properties{
		HealTimeout:                    time.Minute * 1,
		CloseTimeout:                   time.Second * 5,
		HealRequestTimeout:             time.Second * 20,
		HealRequestConnectTimeout:      time.Second * 15,
		HealRequestConnectCheckTimeout: time.Second * 1,
		HealForwarderTimeout:           time.Minute * 1,
		HealRetryCount:                 10,
		HealRetryDelay:                 time.Second * 5,

		// Total DST heal timeout is 20 seconds.
		HealDSTNSEWaitTimeout: time.Second * 30,       // Maximum time to wait for NSMD/NSE to re-appear
		HealDSTNSEWaitTick:    500 * time.Millisecond, // Wait timeout to appear of NSE
		HealEnabled:           true,
	}

	// Parse few Environment variables.
	if os.Getenv(NsmdHealEnabled) == "false" {
		values.HealEnabled = false
	}
	dstWaitTimeout := os.Getenv(NsmdHealDSTWaitTimeout)
	if len(dstWaitTimeout) > 0 {
		logrus.Infof("Override HealDSTWaitTimeout: %s", dstWaitTimeout)
		seconds, err := strconv.ParseInt(dstWaitTimeout, 10, 32)
		if err == nil {
			values.HealDSTNSEWaitTimeout = time.Second * time.Duration(seconds)
		} else {
			logrus.Errorf("Failed to parse DST wait timeout value... %v", err)
		}
	}

	retryVal := os.Getenv(NsmdHealRetryCount)
	if retryVal != "" {
		value, err := strconv.ParseInt(retryVal, 10, 32)
		if err != nil {
			logrus.Error(err)
		}
		values.HealRetryCount = int(value)
	}

	return values
}
