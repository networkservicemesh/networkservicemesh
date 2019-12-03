// Copyright (c) 2019 Cisco and/or its affiliates.
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

// +build basic

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestDeleteNSMCr(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running delete NSM Custom Resource test")

	k8s, err := kubetest.NewK8s(g, kubetest.DefaultClear)
	g.Expect(err).To(BeNil())
	defer k8s.Cleanup()

	nodes_setup, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	kubetest.ExpectNSMsCountToBe(k8s, 0, 1)

	logrus.Infof("Deleting NSMD")
	k8s.DeletePods(nodes_setup[0].Nsmd)

	kubetest.ExpectNSMsCountToBe(k8s, 1, 0)
}
