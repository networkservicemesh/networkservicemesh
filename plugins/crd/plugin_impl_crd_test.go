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
	"reflect"
	"testing"

	"github.com/ligato/networkservicemesh/netmesh/model/netmesh"
	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	networkservicemesh "github.com/ligato/networkservicemesh/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func setupEnv(k8s *kubernetes.Clientset, apiextClient *apiextcs.Clientset) error {
	crds := []struct {
		name     string
		plural   string
		fullname string
	}{
		{name: reflect.TypeOf(v1.NetworkServiceEndpoint{}).Name(), plural: v1.NSMEPPlural, fullname: v1.FullNSMEPName},
		{name: reflect.TypeOf(v1.NetworkServiceChannel{}).Name(), plural: v1.NSMChannelPlural, fullname: v1.FullNSMChannelName},
		{name: reflect.TypeOf(v1.NetworkService{}).Name(), plural: v1.NSMPlural, fullname: v1.FullNSMName},
	}
	plugin := Plugin{
		k8sClientset: k8s,
		apiclientset: apiextClient,
	}
	// Setting up testing namespace
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsmTestNamespace,
		},
	}
	if _, err := k8s.CoreV1().Namespaces().Create(&namespace); err != nil {
		return err
	}
	// Creating CRD definitions
	for _, crd := range crds {
		err := createCRD(&plugin, crd.fullname,
			v1.NSMGroup,
			v1.NSMGroupVersion,
			crd.plural,
			crd.name)
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanupEnv(k8s *kubernetes.Clientset, apiextClient *apiextcs.Clientset) error {

	crds := []struct {
		name     string
		plural   string
		fullname string
	}{
		{name: reflect.TypeOf(v1.NetworkServiceEndpoint{}).Name(), plural: v1.NSMEPPlural, fullname: v1.FullNSMEPName},
		{name: reflect.TypeOf(v1.NetworkServiceChannel{}).Name(), plural: v1.NSMChannelPlural, fullname: v1.FullNSMChannelName},
		{name: reflect.TypeOf(v1.NetworkService{}).Name(), plural: v1.NSMPlural, fullname: v1.FullNSMName},
	}
	for _, crd := range crds {
		err := apiextClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.fullname, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	if err := k8s.CoreV1().Namespaces().Delete(nsmTestNamespace, &metav1.DeleteOptions{}); err != nil {
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
		t.Errorf("Fail to get k8s client with error: %+v", err)
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
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nsm-service-1",
					Namespace: nsmTestNamespace,
				},
				Spec: netmesh.NetworkService{
					Name: "nsm-service-1",
					Uuid: "81a66881-4052-46d3-9890-742da5a04b70",
				},
			},
			expectFail: false,
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
}
