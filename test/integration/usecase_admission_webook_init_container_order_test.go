// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	"github.com/onsi/gomega"
)

func TestAdmissionWebhookInitContainerOrderTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	assert := gomega.NewWithT(t)
	k8s, err := kubetest.NewK8s(assert, true)
	defer k8s.Cleanup()
	assert.Expect(err).To(gomega.BeNil())

	awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace(), defaultTimeout)
	defer kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	assert.Expect(err).To(gomega.BeNil())
	defer k8s.SaveTestArtifacts(t)

	kubetest.DeployICMP(k8s, nodes[0].Node, "nse", defaultTimeout)
	nsc := pods.NSCPodWebhook("nsc", nodes[0].Node)
	nsc.Spec.InitContainers = []v1.Container{{
		Name:            "tail-container",
		Image:           "alpine:latest",
		ImagePullPolicy: v1.PullIfNotPresent,
		Command: []string{
			"ifconfig",
		},
	}}
	k8s.CreatePod(nsc)
	k8s.WaitLogsContains(nsc, "tail-container", "nsm", defaultTimeout)
}
