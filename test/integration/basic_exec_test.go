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

// +build single_cluster_suite

package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestExec(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8sWithoutRoles(g, kubetest.ReuseNSMResources)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())
	defer k8s.ProcessArtifacts(t)

	k8s.DeletePodsByName("alpine-pod")

	alpinePod := k8s.CreatePod(pods.AlpinePod("alpine-pod", nil))

	ipResponse, errResponse, error := k8s.Exec(alpinePod, alpinePod.Spec.Containers[0].Name, "ip", "addr")
	g.Expect(error).To(BeNil())
	g.Expect(errResponse).To(Equal(""))
	logrus.Printf("NSC IP status:%s", ipResponse)
	logrus.Printf("End of test")
	k8s.DeletePods(alpinePod)
}
