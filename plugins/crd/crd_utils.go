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

	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	cfg crdutils.Config
)

func newCustomResourceDefinition(plugin *Plugin, FullName, Group, Version, Plural, Name string) error {
	flagset := flag.NewFlagSet(Name, flag.ExitOnError)
	flagset.Var(&cfg.Labels, "labels", "Labels")

	crd := crdutils.NewCustomResourceDefinition(crdutils.Config{
		SpecDefinitionName:    FullName,
		EnableValidation:      true,
		Labels:                crdutils.Labels{LabelsMap: cfg.Labels.LabelsMap},
		ResourceScope:         string(apiextv1beta1.NamespaceScoped),
		Group:                 Group,
		Kind:                  Name,
		Version:               Version,
		Plural:                Plural,
		GetOpenAPIDefinitions: v1.GetOpenAPIDefinitions,
	})

	crdClient := plugin.apiclientset.ApiextensionsV1beta1().CustomResourceDefinitions()

	// First check if the CRD already exists
	oldCRD, err := crdClient.Get(crd.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		plugin.Log.Errorf("error getting CRD %s, type %s", crd.Name, crd.Spec.Names.Kind)
		return err
	}
	if apierrors.IsNotFound(err) {
		// If the CRD does not exist, try to create it
		if _, err := crdClient.Create(crd); err != nil {
			plugin.Log.Errorf("error creating CRD %s, type %s", crd.Name, crd.Spec.Names.Kind)
			return err
		}
		plugin.Log.Infof("created CRD %s, type %s", crd.Name, crd.Spec.Names.Kind)
	}
	if err == nil {
		// Now we try to update the CRD
		crd.ResourceVersion = oldCRD.ResourceVersion
		if _, err := crdClient.Update(crd); err != nil {
			plugin.Log.Errorf("error updating CRD %s, type %s", crd.Name, crd.Spec.Names.Kind)
			return err
		}
		plugin.Log.Infof("updated CRD %s, type %s", crd.Name, crd.Spec.Names.Kind)
	}

	return nil
}
