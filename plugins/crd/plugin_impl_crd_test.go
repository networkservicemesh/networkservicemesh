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

// //go:generate protoc -I ./model/pod --go_out=plugins=grpc:./model/pod ./model/pod/pod.proto

package netmeshplugincrd

import (
	"flag"
	"testing"

	"github.com/ligato/networkservicemesh/netmesh/model/netmesh"
	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	networkservicemesh "github.com/ligato/networkservicemesh/pkg/client/clientset/versioned"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	corev1 "k8s.io/api/core/v1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	nsmTestNamespace = "networkservicemesh-test"
)

var kubeconfig string

func init() {
	kc := flag.String("kube-config", "", "Full path to k8s' cluster kubeconfig file.")
	flag.Parse()
	kubeconfig = *kc
}

func k8sClient(kc string) (*kubernetes.Clientset, *apiextcs.Clientset, *networkservicemesh.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		return nil, nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	apiextClient, err := apiextcs.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	crdClient, err := networkservicemesh.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	return kubeClient, apiextClient, crdClient, nil
}

func setupEnv(k8s *kubernetes.Clientset, _ *apiextcs.Clientset) error {
	// Setting up testing namespace
	namespace := corev1.Namespace{
		ObjectMeta: meta.ObjectMeta{
			Name: nsmTestNamespace,
		},
	}
	_, err := k8s.CoreV1().Namespaces().Get(namespace.ObjectMeta.Name, meta.GetOptions{})
	if err == nil {
		return nil
	}

	if _, err := k8s.CoreV1().Namespaces().Create(&namespace); err != nil {
		return err
	}

	return nil
}

func cleanupEnv(k8s *kubernetes.Clientset, _ *apiextcs.Clientset) error {

	_, err := k8s.CoreV1().Namespaces().Get(nsmTestNamespace, meta.GetOptions{})
	if apierrors.IsNotFound(err) {
		// If not, done
		return nil
	}
	if err := k8s.CoreV1().Namespaces().Delete(nsmTestNamespace, &meta.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func TestCRDValidation(t *testing.T) {

	if kubeconfig == "" {
		t.Skip("This test requires a valid kubeconfig file, skipping...")
	}
	k8sClient, apiextClient, crdClient, err := k8sClient(kubeconfig)
	if err != nil {
		t.Skipf("Fail to get k8s client with error: %+v", err)
	}
	if err := setupEnv(k8sClient, apiextClient); err != nil {
		t.Errorf("Fail to setup test environment with error: %+v", err)
	}
	defer func() {
		if err := cleanupEnv(k8sClient, apiextClient); err != nil {
			t.Errorf("Fail to cleanup test environment with error: %+v", err)
		}
	}()

	testsNS := []struct {
		testName   string
		ns         v1.NetworkService
		expectFail bool
	}{
		{
			testName: "Network Service All Good",
			ns: v1.NetworkService{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-service-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkService{
					Metadata: &common.Metadata{
						Name:      "nsm-service-1",
						Namespace: "nsm-service-1-namespace",
					},
				},
			},
			expectFail: false,
		},
		{
			testName: "Network Service incorrect name",
			ns: v1.NetworkService{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-serv%ice-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkService{
					Metadata: &common.Metadata{
						Name:      "nsm-serv%ice-1",
						Namespace: "nsm-service-1-namespace",
					},
				},
			},
			expectFail: true,
		},
		{
			testName: "Network Service incorrect namespace",
			ns: v1.NetworkService{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-service-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkService{
					Metadata: &common.Metadata{
						Name:      "nsm-service-1",
						Namespace: "nsm-service-1-$namespace",
					},
				},
			},
			expectFail: true,
		},
	}
	for _, test := range testsNS {
		_, err := crdClient.Networkservice().NetworkServices(nsmTestNamespace).Create(&test.ns)
		if err != nil {
			if !test.expectFail {
				t.Errorf("Test '%s' is supposed to succeed but fail with error: %+v", test.testName, err)
				continue
			}
		} else {
			if test.expectFail {
				t.Errorf("Test '%s' is supposed to fail but succeeded.", test.testName)
				continue
			}
		}
	}

	testsEP := []struct {
		testName   string
		ns         v1.NetworkServiceEndpoint
		expectFail bool
	}{
		{
			testName: "Network Service Endpoint All Good",
			ns: v1.NetworkServiceEndpoint{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-service-endpoint-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkServiceEndpoint{
					Metadata: &common.Metadata{
						Name:      "nsm-service-endpoint-1",
						Namespace: "nsm-service-endpoint-1-namespace",
					},
				},
			},
			expectFail: false,
		},
		{
			testName: "Network Service Endpoint incorrect name",
			ns: v1.NetworkServiceEndpoint{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-serv%ice-endpoint-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkServiceEndpoint{
					Metadata: &common.Metadata{
						Name:      "nsm-service-%endpoint-1",
						Namespace: "nsm-service-endpoint-1-namespace",
					},
				},
			},
			expectFail: true,
		},
		{
			testName: "Network Service Endpoint incorrect namespace",
			ns: v1.NetworkServiceEndpoint{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-service-endpoint-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkServiceEndpoint{
					Metadata: &common.Metadata{
						Name:      "nsm-service-endpoint-1",
						Namespace: "nsm-service-endpoint-1-name$space",
					},
				},
			},
			expectFail: true,
		},
	}
	for _, test := range testsEP {
		_, err := crdClient.Networkservice().NetworkServiceEndpoints(nsmTestNamespace).Create(&test.ns)
		if err != nil {
			if !test.expectFail {
				t.Errorf("Test '%s' is supposed to succeed but fail with error: %+v", test.testName, err)
				continue
			}
		} else {
			if test.expectFail {
				t.Errorf("Test '%s' is supposed to fail but succeeded.", test.testName)
				continue
			}
		}
	}

	testsCH := []struct {
		testName   string
		ns         v1.NetworkServiceChannel
		expectFail bool
	}{
		{
			testName: "Network Service Channel All Good",
			ns: v1.NetworkServiceChannel{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-service-channel-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkServiceChannel{
					Metadata: &common.Metadata{
						Name:      "nsm-service-channel-1",
						Namespace: "nsm-service-channel-1-namespace",
					},
				},
			},
			expectFail: false,
		},
		{
			testName: "Network Service Channel incorrect name",
			ns: v1.NetworkServiceChannel{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-serv%ice-channel-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkServiceChannel{
					Metadata: &common.Metadata{
						Name:      "nsm-service-c%h&annel-1",
						Namespace: "nsm-service-channel-1-namespace",
					},
				},
			},
			expectFail: true,
		},
		{
			testName: "Network Service Channel incorrect namespace",
			ns: v1.NetworkServiceChannel{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-service-channel-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkServiceChannel{
					Metadata: &common.Metadata{
						Name:      "nsm-service-channel-1",
						Namespace: "nsm-service-cha&nnel-1-namespace",
					},
				},
			},
			expectFail: true,
		},
		{
			testName: "Network Service Channel incorrect Payload",
			ns: v1.NetworkServiceChannel{
				ObjectMeta: meta.ObjectMeta{
					Name:      "nsm-service-channel-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkServiceChannel{
					Metadata: &common.Metadata{
						Name:      "nsm-service-channel-1",
						Namespace: "nsm-service-channel-1-namespace",
					},
					Payload: "IP%v4%",
				},
			},
			expectFail: true,
		},
	}
	for _, test := range testsCH {
		_, err := crdClient.Networkservice().NetworkServiceChannels(nsmTestNamespace).Create(&test.ns)
		if err != nil {
			if !test.expectFail {
				t.Errorf("Test '%s' is supposed to succeed but fail with error: %+v", test.testName, err)
				continue
			}
		} else {
			if test.expectFail {
				t.Errorf("Test '%s' is supposed to fail but succeeded.", test.testName)
				continue
			}
		}
	}

}
