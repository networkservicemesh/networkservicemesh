// Copyright 2019 VMware, Inc.
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

package crds

import (
	"os"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	nsapiv1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	nscrd "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
)

type NSCRD struct {
	clientset nscrd.Interface
	namespace string
	resource  string
	config    *rest.Config
}

func (nscrd *NSCRD) Create(obj *nsapiv1.NetworkService) (*nsapiv1.NetworkService, error) {
	var result nsapiv1.NetworkService
	err := nscrd.clientset.NetworkserviceV1alpha1().RESTClient().Post().
		Namespace(nscrd.namespace).Resource(nscrd.resource).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (nscrd *NSCRD) Update(obj *nsapiv1.NetworkService) (*nsapiv1.NetworkService, error) {
	var result nsapiv1.NetworkService
	err := nscrd.clientset.NetworkserviceV1alpha1().RESTClient().Put().
		Namespace(nscrd.namespace).Resource(nscrd.resource).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (nscrd *NSCRD) Delete(name string, options *metaV1.DeleteOptions) error {
	return nscrd.clientset.NetworkserviceV1alpha1().RESTClient().Delete().
		Namespace(nscrd.namespace).Resource(nscrd.resource).
		Name(name).Body(options).Do().
		Error()
}

func (nscrd *NSCRD) Get(name string) (*nsapiv1.NetworkService, error) {
	var result nsapiv1.NetworkService
	err := nscrd.clientset.NetworkserviceV1alpha1().RESTClient().Get().
		Namespace(nscrd.namespace).Resource(nscrd.resource).
		Name(name).Do().Into(&result)
	return &result, err
}

// NewNSCRD creates a new Clientset for the default config.
func NewNSCRD(namespace string) (*NSCRD, error) {
	path := os.Getenv("KUBECONFIG")
	if len(path) == 0 {
		path = os.Getenv("HOME") + "/.kube/config"
	}

	return NewNSCRDWithConfig(namespace, path)
}

// NewNSCRDWithConfig creates a new Clientset for the given config.
func NewNSCRDWithConfig(namespace, kubepath string) (*NSCRD, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubepath)
	if err != nil {
		return nil, err
	}

	nsmNamespace := namespace
	client := NSCRD{
		namespace: nsmNamespace,
		resource:  "networkservices",
		config:    config,
	}
	client.clientset, err = nscrd.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &client, nil
}
