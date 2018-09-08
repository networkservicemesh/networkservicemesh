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

	"github.com/Masterminds/semver"
	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nsmCRDVersion              = "0.0.1"
	nsmCRDVersionAnnotationKey = "networkservicemesh.io/nsm-crd-version"
	nsmObjectUpdateRetries     = 5
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

	crdClient := plugin.apiclientset
	// Add NSM CRD version
	crd.ObjectMeta.Annotations = map[string]string{
		nsmCRDVersionAnnotationKey: nsmCRDVersion,
	}
	// Starting with 1.11.X SpecReplicasPath and StatusReplicasPath have become mandatory fields,
	// but since NewCustomResourceDefinition function does not set them, CRD creation fails in > 1.11.0 k8s clusters.
	// As a workaround, setting these two fields manually here.
	crd.Spec.Subresources.Scale.SpecReplicasPath = ".spec.replicas"
	crd.Spec.Subresources.Scale.StatusReplicasPath = ".status.replicas"
	if err := createCRDObject(crd, crdClient); err != nil {
		plugin.Log.Errorf("fail to create CRD with error: %#v", err)
		return err
	}

	return nil
}

func createCRDObject(newCRD *apiextv1beta1.CustomResourceDefinition, crdClient *apiextcs.Clientset) error {
	// First check if the CRD already exists
	oldCRD, err := crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(newCRD.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("error getting CRD %s, type %s with error: %+v", newCRD.ObjectMeta.Name, newCRD.Spec.Names.Kind, err)
	}
	if apierrors.IsNotFound(err) {
		// If the CRD does not exist, try to create it
		if _, err := crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(newCRD); err != nil {
			return fmt.Errorf("fail creating CRD %s, type %s with error: %#v", newCRD.ObjectMeta.Name, newCRD.Spec.Names.Kind, err)
		}
		return nil
	}
	// Check if CRD has the version annotation, if not, then it means it is old CRD
	// and we update it uncoditionally.
	version, ok := oldCRD.ObjectMeta.Annotations[nsmCRDVersionAnnotationKey]
	if !ok {
		// Exisiting CRD does not have the version annotation, updating it to new definition
		// uncoditionally
		return updateCRD(newCRD, crdClient)
	}
	// Existing CRD has version info, next check is to see if existing CRD version is "<" or "==" or ">"
	// if "<" than new CRD, it will be updated, if "==", then no action , if ">" than new CRD, then we will fail as
	// it possible, that older version of NSM controller started on a cluster with newer CRD definitions.
	existingVersion, err := semver.NewVersion(version)
	if err != nil {
		// Since we failed to process existing CRD version, then we update CRD attempting to bring it to the right level
		return updateCRD(newCRD, crdClient)
	}
	newVersion, _ := semver.NewVersion(newCRD.ObjectMeta.Annotations[nsmCRDVersionAnnotationKey])
	if existingVersion.LessThan(newVersion) {
		// It is upgrade case, updating CRD to the new CRD version
		return updateCRD(newCRD, crdClient)
	}
	if existingVersion.GreaterThan(newVersion) {
		// Downgrade scenario, we have to fail and let the user to resolve this inconsistency
		return fmt.Errorf("fail creating CRD %s, as desired version %s is lower than already exisiting CRD object version %s",
			newCRD.Name, newVersion.String(), existingVersion.String())
	}
	// Old CRD version "==" to new CRD version, do nothing
	return nil
}

// updateCRD attempts to update existing CRD with new definitions. number of attempts is defined in
// nsmObjectUpdateRetries. In case of a conflict error code is returned, the update is re-attempted
// as per optimistic concurrency approach.
func updateCRD(newCRD *apiextv1beta1.CustomResourceDefinition, crdClient *apiextcs.Clientset) error {
	for i := 0; i < nsmObjectUpdateRetries; i++ {
		oldCRD, err := crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(newCRD.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("update: error getting CRD %s, type %s with error: %+v", newCRD.ObjectMeta.Name, newCRD.Spec.Names.Kind, err)
		}
		newCRD.ResourceVersion = oldCRD.ResourceVersion
		_, err = crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Update(newCRD)
		if err == nil {
			return nil
		} else if !apierrors.IsConflict(err) {
			return fmt.Errorf("update: fail updating CRD %s, type %s with error: %#v", newCRD.ObjectMeta.Name, newCRD.Spec.Names.Kind, err)
		}
	}
	return fmt.Errorf("update: fail to update CRD after %d retries", nsmObjectUpdateRetries)
}
