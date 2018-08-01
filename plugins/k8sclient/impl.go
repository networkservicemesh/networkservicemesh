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
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ligato/networkservicemesh/utils/command"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	"github.com/ligato/networkservicemesh/utils/helper/plugintools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"github.com/ligato/networkservicemesh/utils/registry"
)

// Plugin for k8sclient
type Plugin struct {
	idempotent.Impl
	Deps

	k8sClientConfig *rest.Config
	k8sClientset    *kubernetes.Clientset
}

// Init Plugin
func (p *Plugin) Init() error {
	return p.Impl.IdempotentInit(plugintools.LoggingInitFunc(p.Log, p, p.init))
}

func (p *Plugin) init() error {
	var err error

	p.KubeConfig = command.RootCmd().Flags().Lookup(KubeConfigFlagName).Value.String()

	p.Log.WithField("kubeconfig", p.KubeConfig).Info("Loading kubernetes client config")
	p.k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", p.KubeConfig)
	if err != nil {
		return fmt.Errorf("Failed to build kubernetes client config: %s", err)
	}

	p.k8sClientset, err = kubernetes.NewForConfig(p.k8sClientConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %s", err)
	}

	return nil
}

// Close Plugin
func (p *Plugin) Close() error {
	return p.Impl.IdempotentClose(plugintools.LoggingCloseFunc(p.Log, p, p.close))
}

func (p *Plugin) close() error {
	registry.Shared().Delete(p)
	return deptools.Close(p)
}

// GetClientConfig returns a pointer to our rest.Config object
func (p *Plugin) GetClientConfig() *rest.Config {
	return p.k8sClientConfig
}

// GetClientset returns a pointer to our kubernetes.Clientset object
func (p *Plugin) GetClientset() *kubernetes.Clientset {
	return p.k8sClientset
}
