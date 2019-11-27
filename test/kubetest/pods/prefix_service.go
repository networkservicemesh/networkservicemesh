// Copyright (c) 2019 Cisco Systems, Inc.
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

package pods

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/prefixcollector"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

// PrefixServicePod creates pod that collects cluster network prefixes and stores them to ConfigMap
func PrefixServicePod(nmsp string) *v1.Pod {
	const name = "prefix-service"

	envVars := []v1.EnvVar{
		{
			Name:  namespace.NsmNamespaceEnv,
			Value: nmsp,
		},
	}
	if prefixes := utils.EnvVar(prefixcollector.ExcludedPrefixesEnv).StringValue(); prefixes != "" {
		envVars = append(envVars, v1.EnvVar{
			Name:  prefixcollector.ExcludedPrefixesEnv,
			Value: prefixes,
		})

	}

	return &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			ServiceAccountName: NSMgrServiceAccount,
			Containers: []v1.Container{
				containerMod(&v1.Container{
					Name:            name,
					Image:           "networkservicemesh/prefix-service",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         []string{"/bin/prefix-service"},
					Env:             envVars,
				}),
			},
			TerminationGracePeriodSeconds: &ZeroGraceTimeout,
		},
	}
}
