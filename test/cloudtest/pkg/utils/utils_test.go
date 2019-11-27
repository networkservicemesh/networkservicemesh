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
	"testing"

	"github.com/onsi/gomega"
)

func TestMatchPattern(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(MatchRetestPattern([]string{
		"unable to establish connection to VPP (VPP API socket file /run/vpp/api.sock does not exist)",
	}, "time=\"2019-11-22 09:28:45.55766\" level=fatal msg=\"unable to establish connection to VPP (VPP API socket file /run/vpp/api.sock does not exist)\" loc=\"vpp-agent/main.go(65)\" logger=defaultLogger")).To(gomega.Equal(true))
}
