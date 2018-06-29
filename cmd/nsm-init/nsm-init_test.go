// Copyright (c) 2018 Cisco and/or its affiliates.
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
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestParseNSMClientConfig(t *testing.T) {
	tests := []struct {
		testName                string
		configMapName           *v1.ConfigMap
		expectedNetworkServices int
		expectError             bool
	}{
		{
			testName: "Good configmap",
			configMapName: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nsm-1-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"networkService": `    
- name: network-service-1
  serviceInterface:
  - type: 0
    preference: 1
    metadata:
      name: interface-1
  - type: 2
    preference: 3
    metadata:
      name: interface-2
- name: network-service-2
  serviceInterface:
  - type: 0
    preference: 1
    metadata:
      name: interface-3`,
				},
			},
			expectedNetworkServices: 2,
			expectError:             false,
		},
		{
			testName: "Bad configmap",
			configMapName: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nsm-1-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"wrongTag": `    
- name: network-service-1
  serviceInterface:
  - type: 0
    preference: 1
    metadata:
      name: interface-1
  - type: 2
    preference: 3
    metadata:
      name: interface-2`,
				},
			},
			expectedNetworkServices: 0,
			expectError:             true,
		},
	}

	client := fake.NewSimpleClientset()

	for _, test := range tests {
		if err := createConfigMap(client, test.configMapName); err != nil {
			t.Fatalf("Failed to build configmap(s) with error: %v", err)
		}
		cm, err := checkClientConfigMap(test.configMapName.ObjectMeta.Name, test.configMapName.ObjectMeta.Namespace, client)
		if err != nil {
			t.Fatalf("Failed to access configmap with error: %v", err)
		}
		ns, err := parseConfigMap(cm)
		if err != nil && !test.expectError {
			t.Fatalf("Failed to parse configmap with error: %v", err)
		}
		if len(ns) != test.expectedNetworkServices {
			t.Fatalf("Failed as expected %d number NetworkServiced, but got %d", test.expectedNetworkServices, len(ns))
		}
		deleteConfigMap(client, test.configMapName)
	}
}

func createConfigMap(k8s *fake.Clientset, configMapName *v1.ConfigMap) error {
	_, err := k8s.CoreV1().ConfigMaps(configMapName.ObjectMeta.Namespace).Create(configMapName)
	return err
}

func deleteConfigMap(k8s *fake.Clientset, configMapName *v1.ConfigMap) error {
	zero := int64(0)
	return k8s.CoreV1().ConfigMaps(configMapName.ObjectMeta.Namespace).Delete(configMapName.ObjectMeta.Name, &metav1.DeleteOptions{GracePeriodSeconds: &zero})
}
