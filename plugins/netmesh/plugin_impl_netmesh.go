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

package netmesh

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/datasync/kvdbsync"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/health/statuscheck/model/status"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/networkservicemesh/nsmdp"
)

// Plugin watches K8s resources and causes all changes to be reflected in the ETCD
// data store.
type Plugin struct {
	Deps

	stopCh chan struct{}
	wg     sync.WaitGroup

	k8sClientConfig *rest.Config
	k8sClientset    *kubernetes.Clientset

	StatusMonitor statuscheck.StatusReader
	etcdMonitor   EtcdMonitor
}

// EtcdMonitor defines the state data for the Etcd Monitor
type EtcdMonitor struct {
	// Operational status is the last seen operational status from the
	// plugin monitor
	status status.OperationalState
	// lastRev is the last seen revision of the plugin's status in the
	// data store
	lastRev int64

	// broker is the interface to a key-val data store.
	broker KeyProtoValBroker
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	local.PluginInfraDeps
	// Kubeconfig with k8s cluster address and access credentials to use.
	KubeConfig config.PluginConfig
	// broker is used to propagate changes into a key-value datastore.
	// contiv-netmesh uses ETCD as datastore.
	Publish *kvdbsync.Plugin
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (plugin *Plugin) Init() error {
	var err error
	plugin.Log.SetLevel(logging.DebugLevel)
	plugin.stopCh = make(chan struct{})

	kubeconfig := plugin.KubeConfig.GetConfigName()
	plugin.Log.WithField("kubeconfig", kubeconfig).Info("Loading kubernetes client config")
	plugin.k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client config: %s", err)
	}

	plugin.k8sClientset, err = kubernetes.NewForConfig(plugin.k8sClientConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %s", err)
	}

	netmeshPrefix := plugin.Publish.ServiceLabel.GetAgentPrefix()

	plugin.etcdMonitor.broker = plugin.Publish.Deps.KvPlugin.NewBroker(netmeshPrefix)
	plugin.etcdMonitor.status = status.OperationalState_INIT
	plugin.etcdMonitor.lastRev = 0

	return nil
}

// AfterInit starts all reflectors. They have to be started in AfterInit so that
// the kvdbsync is fully initialized and ready for publishing when a k8s
// notification comes.
func (plugin *Plugin) AfterInit() error {
	go plugin.monitorEtcdStatus(plugin.stopCh)

	dp := nsmdp.NewNSMDevicePlugin()
	dp.Serve()

	return nil
}

// Close stops all reflectors.
func (plugin *Plugin) Close() error {
	close(plugin.stopCh)
	plugin.wg.Wait()
	return nil
}

// monitorEtcdStatus monitors the KSR's connection to the Etcd Data Store.
func (plugin *Plugin) monitorEtcdStatus(closeCh chan struct{}) {
	for {
		select {
		case <-closeCh:
			plugin.Log.Info("Closing")
			return
		case <-time.After(1 * time.Second):
			sts := plugin.StatusMonitor.GetAllPluginStatus()
			for k, v := range sts {
				if k == "etcdv3" {
					plugin.etcdMonitor.processEtcdMonitorEvent(v.State)
					plugin.etcdMonitor.checkEtcdTransientError()
					break
				}
			}
		}
	}
}

// processEtcdMonitorEvent processes ectd plugin's status events and, if an
// Etcd problem is detected, generates a resync event for all reflectors.
func (etcdm *EtcdMonitor) processEtcdMonitorEvent(ns status.OperationalState) {
	switch ns {
	case status.OperationalState_INIT:
		if etcdm.status == status.OperationalState_OK {
		}
	case status.OperationalState_ERROR:
		if etcdm.status == status.OperationalState_OK {
		}
	case status.OperationalState_OK:
		if etcdm.status == status.OperationalState_INIT ||
			etcdm.status == status.OperationalState_ERROR {
		}
	}
	etcdm.status = ns
}

// checkEtcdTransientError checks if there was a transient error that results
// in data loss in Etcd. If yes, resync of netmesh is triggered.
func (etcdm *EtcdMonitor) checkEtcdTransientError() {

}
