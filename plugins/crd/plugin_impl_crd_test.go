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

package crd

import (
	"flag"
	"fmt"
	"testing"
	"time"

	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	networkservicemesh "github.com/ligato/networkservicemesh/pkg/client/clientset/versioned"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
	corev1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	nsmTestNamespace     = "networkservicemesh-test"
	crdCompletionTimeout = 2 * time.Second
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

func TestCRValidation(t *testing.T) {

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
				Spec: &netmesh.NetworkService{
					NetworkServiceName: "nsm-service-1",
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
				Spec: &netmesh.NetworkService{
					NetworkServiceName: "nsm-serv%ice-1",
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
				Spec: &netmesh.NetworkService{
					NetworkServiceName: "nsm-service-1",
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
				Spec: &netmesh.NetworkServiceEndpoint{
					NetworkServiceName: "nsm-service-endpoint-1",
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
				Spec: &netmesh.NetworkServiceEndpoint{
					NetworkServiceName: "nsm-service-%endpoint-1",
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
				Spec: &netmesh.NetworkServiceEndpoint{
					NetworkServiceName: "nsm-service-endpoint-1",
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
}

type crdTest struct {
	testName         string
	oldCRD           *apiextv1beta1.CustomResourceDefinition
	createOld        bool
	newCRD           *apiextv1beta1.CustomResourceDefinition
	newCRDAnnotation map[string]string
	oldCRDAnnotation map[string]string
	shouldFail       bool
}

type crdParameters struct {
	fullName     string
	group        string
	groupVersion string
	plural       string
	typeName     string
}

func TestCreateCRDObject(t *testing.T) {
	crds := []crdParameters{
		{
			fullName:     v1.FullNSMEPName,
			group:        v1.NSMGroup,
			groupVersion: v1.NSMGroupVersion,
			plural:       v1.NSMEPPlural,
			typeName:     v1.NSMEPTypeName,
		},
		{
			fullName:     v1.FullNSMName,
			group:        v1.NSMGroup,
			groupVersion: v1.NSMGroupVersion,
			plural:       v1.NSMPlural,
			typeName:     v1.NSMTypeName,
		},
	}
	tests := []crdTest{
		{
			testName:  "Simple CRD creation",
			createOld: false,
			newCRDAnnotation: map[string]string{
				nsmCRDVersionAnnotationKey: "0.0.1",
			},
			shouldFail: false,
		},
		{
			testName:  "old CRD version < then new CRD version",
			createOld: true,
			oldCRDAnnotation: map[string]string{
				nsmCRDVersionAnnotationKey: "0.0.1",
			},
			newCRDAnnotation: map[string]string{
				nsmCRDVersionAnnotationKey: "0.0.2",
			},
			shouldFail: false,
		},
		{
			testName:  "old CRD version == new CRD version",
			createOld: true,
			oldCRDAnnotation: map[string]string{
				nsmCRDVersionAnnotationKey: "0.0.1",
			},
			newCRDAnnotation: map[string]string{
				nsmCRDVersionAnnotationKey: "0.0.1",
			},
			shouldFail: false,
		},
		{
			testName:  "old CRD version > new CRD version",
			createOld: true,
			oldCRDAnnotation: map[string]string{
				nsmCRDVersionAnnotationKey: "0.0.2",
			},
			newCRDAnnotation: map[string]string{
				nsmCRDVersionAnnotationKey: "0.0.1",
			},
			shouldFail: true,
		},
	}

	if kubeconfig == "" {
		t.Skip("This test requires a valid kubeconfig file, skipping...")
	}
	k8sClient, apiextClient, _, err := k8sClient(kubeconfig)
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

	for _, crd := range crds {
		for _, test := range tests {
			if test.createOld {
				test.oldCRD = crdutils.NewCustomResourceDefinition(crdutils.Config{
					SpecDefinitionName:    crd.fullName,
					EnableValidation:      true,
					Labels:                crdutils.Labels{LabelsMap: cfg.Labels.LabelsMap},
					ResourceScope:         string(apiextv1beta1.NamespaceScoped),
					Group:                 crd.group,
					Kind:                  crd.typeName,
					Version:               crd.groupVersion,
					Plural:                crd.plural,
					GetOpenAPIDefinitions: v1.GetOpenAPIDefinitions,
				})
				test.oldCRD.Spec.Subresources.Scale.SpecReplicasPath = ".spec.replicas"
				test.oldCRD.Spec.Subresources.Scale.StatusReplicasPath = ".status.replicas"
				test.oldCRD.ObjectMeta.Annotations = test.oldCRDAnnotation
				if err := createCRDObject(test.oldCRD, apiextClient); err != nil {
					t.Fatalf("test %s failed with error: %s, CRD name: %s CRD kind: %s", test.testName, err.Error(), crd.fullName, crd.typeName)
				}
				if err := waitForCRDCreate(apiextClient, test.oldCRD, crdCompletionTimeout); err != nil {
					t.Fatalf("waitForCRD %s failed with error: %s, CRD name: %s CRD kind: %s", test.testName, err.Error(), crd.fullName, crd.typeName)
				}
			}
			test.newCRD = crdutils.NewCustomResourceDefinition(crdutils.Config{
				SpecDefinitionName:    crd.fullName,
				EnableValidation:      true,
				Labels:                crdutils.Labels{LabelsMap: cfg.Labels.LabelsMap},
				ResourceScope:         string(apiextv1beta1.NamespaceScoped),
				Group:                 crd.group,
				Kind:                  crd.typeName,
				Version:               crd.groupVersion,
				Plural:                crd.plural,
				GetOpenAPIDefinitions: v1.GetOpenAPIDefinitions,
			})
			test.newCRD.Spec.Subresources.Scale.SpecReplicasPath = ".spec.replicas"
			test.newCRD.Spec.Subresources.Scale.StatusReplicasPath = ".status.replicas"
			test.newCRD.ObjectMeta.Annotations = test.newCRDAnnotation
			err := createCRDObject(test.newCRD, apiextClient)
			if err != nil {
				if !test.shouldFail {
					t.Fatalf("test %s failed with error: %s, CRD name: %s CRD kind: %s", test.testName, err.Error(), crd.fullName, crd.typeName)
				}
			}
			if err := waitForCRDCreate(apiextClient, test.newCRD, crdCompletionTimeout); err != nil {
				t.Fatalf("waitForCRD %s failed with error: %s, CRD name: %s CRD kind: %s", test.testName, err.Error(), crd.fullName, crd.typeName)
			}
			// Test was successful, need to clean up for the next test
			if err := apiextClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(test.newCRD.ObjectMeta.Name,
				&meta.DeleteOptions{}); err != nil {
				t.Fatalf("failed to clean up CRD %s with error: %+v", crd.typeName+"."+crd.group, err)
			}
			if err := waitForCRDDelete(apiextClient, test.newCRD, crdCompletionTimeout); err != nil {
				t.Fatalf("waitForCRDDelete %s failed with error: %s, CRD name: %s CRD kind: %s", test.testName, err.Error(), crd.fullName, crd.typeName)
			}
		}
	}
}

func waitForCRDCreate(crdClient *apiextcs.Clientset, crd *apiextv1beta1.CustomResourceDefinition, timeout time.Duration) error {
	stopCh := time.After(timeout)
	for {
		select {
		case <-stopCh:
			return fmt.Errorf("timeout expired waiting for CRD %s to be created", crd.ObjectMeta.Name)
		default:
			_, err := crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.ObjectMeta.Name, meta.GetOptions{})
			if err == nil {
				return nil
			}
		}
	}
}

func waitForCRDDelete(crdClient *apiextcs.Clientset, crd *apiextv1beta1.CustomResourceDefinition, timeout time.Duration) error {
	stopCh := time.After(timeout)
	for {
		select {
		case <-stopCh:
			return fmt.Errorf("timeout expired waiting for CRD %s to be created", crd.ObjectMeta.Name)
		default:
			_, err := crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.ObjectMeta.Name, meta.GetOptions{})
			if err != nil && apierrors.IsNotFound(err) {
				return nil
			}
		}
	}
}
