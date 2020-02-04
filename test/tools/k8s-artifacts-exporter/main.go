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

package main

import (
	"os"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/artifacts"

	"k8s.io/client-go/kubernetes"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	if !artifacts.NeedToSave() {
		logrus.Warn("Envs are not set. Nothing to save.")
		return
	}
	path := os.Getenv("KUBECONFIG")
	if path == "" {
		path = os.Getenv("HOME") + "/.kube/config"
	}
	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		logrus.Fatal(err.Error())
	}
	namespace := os.Getenv("NSM_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatal(err.Error())
	}
	k8s := kubetest.NewK8sWithClientset(clientset, namespace)
	k8s.SaveArtifacts()
}
