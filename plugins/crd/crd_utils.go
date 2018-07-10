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
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// networkServiceValidation generates OpenAPIV3 validator for NetworkService CRD
func networkServiceValidation() *apiextv1beta1.CustomResourceValidation {
	maxLength := int64(64)
	validation := &apiextv1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextv1beta1.JSONSchemaProps{
			Properties: map[string]apiextv1beta1.JSONSchemaProps{
				"spec": {
					Required: []string{"metadata"},
					Properties: map[string]apiextv1beta1.JSONSchemaProps{
						"metadata": {
							Required: []string{"name"},
							Properties: map[string]apiextv1beta1.JSONSchemaProps{
								"name": {
									Type:        "string",
									MaxLength:   &maxLength,
									Description: "NetworkServiceEndpoints Name",
									Pattern:     `^[a-zA-Z0-9]+[\-a-zA-Z0-9]*$`,
								},
								"namespace": {
									Type:        "string",
									MaxLength:   &maxLength,
									Description: "NetworkServiceEndpoints Namespace",
									Pattern:     `^[a-zA-Z0-9]+[\-a-zA-Z0-9]*$`,
								},
							},
						},
					},
				},
			},
		},
	}
	return validation
}

// networkServiceEndpointsValidation generates OpenAPIV3 validator for NetworkServiceEndpoints CRD
func networkServiceEndpointsValidation() *apiextv1beta1.CustomResourceValidation {
	maxLength := int64(64)
	validation := &apiextv1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextv1beta1.JSONSchemaProps{
			Properties: map[string]apiextv1beta1.JSONSchemaProps{
				"spec": {
					Required: []string{"metadata"},
					Properties: map[string]apiextv1beta1.JSONSchemaProps{
						"metadata": {
							Required: []string{"name"},
							Properties: map[string]apiextv1beta1.JSONSchemaProps{
								"name": {
									Type:        "string",
									MaxLength:   &maxLength,
									Description: "NetworkServiceEndpoints Name",
									Pattern:     `^[a-zA-Z0-9]+[\-a-zA-Z0-9]*$`,
								},
								"namespace": {
									Type:        "string",
									MaxLength:   &maxLength,
									Description: "NetworkServiceEndpoints Namespace",
									Pattern:     `^[a-zA-Z0-9]+[\-a-zA-Z0-9]*$`,
								},
							},
						},
					},
				},
			},
		},
	}
	return validation
}

// networkServiceChannels generates OpenAPIV3 validator for NetworkServiceChannels CRD
func networkServiceChannelsValidation() *apiextv1beta1.CustomResourceValidation {
	maxLength := int64(64)
	validation := &apiextv1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextv1beta1.JSONSchemaProps{
			Properties: map[string]apiextv1beta1.JSONSchemaProps{
				"spec": {
					Required: []string{"metadata"},
					Properties: map[string]apiextv1beta1.JSONSchemaProps{
						"metadata": {
							Required: []string{"name"},
							Properties: map[string]apiextv1beta1.JSONSchemaProps{
								"name": {
									Type:        "string",
									MaxLength:   &maxLength,
									Description: "NetworkServiceEndpoints Name",
									Pattern:     `^[a-zA-Z0-9]+[\-a-zA-Z0-9]*$`,
								},
								"namespace": {
									Type:        "string",
									MaxLength:   &maxLength,
									Description: "NetworkServiceEndpoints Namespace",
									Pattern:     `^[a-zA-Z0-9]+[\-a-zA-Z0-9]*$`,
								},
							},
						},
					},
				},
			},
		},
	}
	return validation
}

// Create the CRD resource, ignore error if it already exists
func createCRD(plugin *Plugin, FullName, Group, Version, Plural, Name string) error {

	var validation *apiextv1beta1.CustomResourceValidation
	switch Name {
	case "NetworkService":
		validation = networkServiceValidation()
	case "NetworkServiceEndpoints":
		validation = networkServiceEndpointsValidation()
	case "NetworkServiceChannels":
		validation = networkServiceChannelsValidation()
	default:
		validation = &apiextv1beta1.CustomResourceValidation{}
	}
	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: meta.ObjectMeta{Name: FullName},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   Group,
			Version: Version,
			Scope:   apiextv1beta1.NamespaceScoped,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural: Plural,
				Kind:   Name,
			},
			Validation: validation,
		},
	}
	_, cserr := plugin.apiclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if apierrors.IsAlreadyExists(cserr) {
		return nil
	}

	return cserr
}
