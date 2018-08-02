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

package k8sclient

import (
	"github.com/ligato/networkservicemesh/plugins/idempotent"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Interface is the interface to a k8sclient plugin
type API interface {
	GetClientConfig() *rest.Config
	GetClientset() *kubernetes.Clientset
}

// PluginAPI for k8sclient
type PluginAPI interface {
	idempotent.PluginAPI
}
