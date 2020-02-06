// Copyright (c) 2020 Cisco and/or its affiliates.
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

// +build usecase

package integration

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestNSCAndICMPRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSCAndICMP(&testNSCAndNSEParams{
		t:               t,
		nodeCount:       2,
		useWebhook:      false,
		disableVHost:    false,
		remoteMechanism: "VXLAN",
		clearOption:     kubetest.DefaultClear,
	})
}

func TestNSCAndICMPRemoteVeth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	testNSCAndICMP(&testNSCAndNSEParams{
		t:               t,
		nodeCount:       2,
		useWebhook:      false,
		disableVHost:    true,
		remoteMechanism: "VXLAN",
		clearOption:     kubetest.DefaultClear,
	})
}
