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

package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

const (
	nsmSRIOVNodeName = "networkservicemesh.io/sriov-node-name"
	// containersConfigPath specifies location where sriov controller stores per POD network service
	// configuration file.
	// TODO (sbezverk) 1. how to clean up after POD which is using this file is gone? The controller could cleanup
	// this folder during a boot up, but how to detect which one is used and which one not?
	containersConfigPath = "/var/lib/networkservicemesh/sriov-controller/config"
)

type operationType int

const (
	operationAdd operationType = iota
	operationDeleteAll
	operationUpdate
	operationDeleteEntry
)

var (
	kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
	wg         sync.WaitGroup
)

type message struct {
	op        operationType
	configMap *v1.ConfigMap
}

// VF describes a single instance of VF
type VF struct {
	NetworkService string `yaml:"networkService" json:"networkService"`
	PCIAddr        string `yaml:"pciAddr" json:"pciAddr"`
	ParentDevice   string `yaml:"parentDevice" json:"parentDevice"`
	VFLocalID      int32  `yaml:"vfLocalID" json:"vfLocalID"`
	VFIODevice     string `yaml:"vfioDevice" json:"vfioDevice"`
}

// VFs is map of ALL found VFs on a specific host kyed by PCI address
type VFs struct {
	vfs map[string]*VF
	sync.RWMutex
}

func newVFs() *VFs {
	v := &VFs{}
	vfs := map[string]*VF{}
	v.vfs = vfs
	return v
}

type configMessage struct {
	op      operationType
	pciAddr string
	vf      VF
}

type configController struct {
	stopCh       chan struct{}
	configCh     chan configMessage
	k8sClientset *kubernetes.Clientset
	informer     cache.SharedIndexInformer
	vfs          *VFs
	nodeName     string
}

func newConfigController() *configController {
	c := configController{}
	vfs := newVFs()
	c.vfs = vfs

	return &c
}

func setupInformer(informer cache.SharedIndexInformer, queue workqueue.RateLimitingInterface) {
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				configMap := obj.(*v1.ConfigMap)
				queue.Add(message{
					op: operationAdd, configMap: configMap})
			},
			UpdateFunc: func(old, new interface{}) {
				configMapNew := new.(*v1.ConfigMap)
				configMapOld := old.(*v1.ConfigMap)
				if configMapNew.ResourceVersion == configMapOld.ResourceVersion {
					return
				}
				queue.Add(message{
					op: operationUpdate, configMap: configMapNew})
			},
			DeleteFunc: func(obj interface{}) {
				configMap := obj.(*v1.ConfigMap)
				queue.Add(message{
					op: operationDeleteAll, configMap: configMap})
			},
		},
	)
}

func initConfigController(cc *configController) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	cc.informer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				// Only List configmap for the local node
				options.LabelSelector = fmt.Sprintf("%s=%s", nsmSRIOVNodeName, cc.nodeName)
				return cc.k8sClientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				// Only Watch configmap for the local node
				options.LabelSelector = fmt.Sprintf("%s=%s", nsmSRIOVNodeName, cc.nodeName)
				return cc.k8sClientset.CoreV1().ConfigMaps(metav1.NamespaceAll).Watch(options)
			},
		},
		&v1.ConfigMap{},
		10*time.Second,
		cache.Indexers{},
	)

	setupInformer(cc.informer, queue)

	go cc.informer.Run(cc.stopCh)
	logrus.Info("Started  config controller shared informer factory.")

	// Wait for the informer caches to finish performing it's initial sync of
	// resources
	if !cache.WaitForCacheSync(cc.stopCh, cc.informer.HasSynced) {
		logrus.Error("Error waiting for informer cache to sync")
	}
	logrus.Info("ConfigController's Informer cache is ready")

	// Read forever from the work queue
	go workforever(cc, queue, cc.stopCh)
}

func workforever(cc *configController, queue workqueue.RateLimitingInterface, stopCh chan struct{}) {
	for {
		obj, shutdown := queue.Get()
		msg := obj.(message)
		// If the queue has been shut down, we should exit the work queue here
		if shutdown {
			logrus.Error("shutdown signaled, closing stopCh")
			close(stopCh)
			return
		}

		func(obj message) {
			defer queue.Done(obj)
			switch obj.op {
			case operationAdd:
				logrus.Infof("Config map add called")
				if err := processConfigMapAdd(cc, obj.configMap); err != nil {
					logrus.Errorf("fail to process add of configmap %s/%s with error: %+v",
						obj.configMap.ObjectMeta.Namespace,
						obj.configMap.ObjectMeta.Name, err)
				}
			case operationUpdate:
				logrus.Info("Config map update called")
				if err := processConfigMapUpdate(cc, obj.configMap); err != nil {
					logrus.Errorf("fail to process update of configmap %s/%s with error: %+v",
						obj.configMap.ObjectMeta.Namespace,
						obj.configMap.ObjectMeta.Name, err)
				}
			case operationDeleteAll:
				logrus.Info("Config map delete called")
				if err := processConfigMapDelete(cc, obj.configMap); err != nil {
					logrus.Errorf("fail to process delete of configmap %s/%s with error: %+v",
						obj.configMap.ObjectMeta.Namespace,
						obj.configMap.ObjectMeta.Name, err)
				}
			}
			queue.Forget(obj)
		}(msg)
	}
}

func processConfigMap(configMap *v1.ConfigMap) (map[string]*VF, error) {
	vfs := map[string]*VF{}
	for k, v := range configMap.Data {
		vf := VF{}
		if err := yaml.Unmarshal([]byte(v), &vf); err != nil {
			return nil, err
		}
		vfs[k] = &vf
	}
	return vfs, nil
}

func processConfigMapAdd(cc *configController, configMap *v1.ConfigMap) error {
	logrus.Infof("Processing Add configmap %s/%s", configMap.ObjectMeta.Namespace, configMap.ObjectMeta.Name)
	vfs, err := processConfigMap(configMap)
	if err != nil {
		return err
	}
	cc.vfs.Lock()
	cc.vfs.vfs = vfs
	cc.vfs.Unlock()
	logrus.Infof("Imported: %d VF configuration(s). Sending configuration to serviceController.", len(cc.vfs.vfs))
	for k, vf := range cc.vfs.vfs {
		cc.configCh <- configMessage{op: operationAdd, pciAddr: k, vf: *vf}
	}

	return nil
}

// diffMap compares maps and returns a third map with missing keys/values
func diffMap(m1, m2 map[string]*VF) map[string]*VF {
	r := map[string]*VF{}
	for k, v := range m1 {
		if _, ok := m2[k]; !ok {
			r[k] = v
		}
	}
	return r
}

func processConfigMapUpdate(cc *configController, configMap *v1.ConfigMap) error {
	logrus.Infof("Processing Update configmap %s/%s", configMap.ObjectMeta.Namespace, configMap.ObjectMeta.Name)
	vfs, err := processConfigMap(configMap)
	if err != nil {
		return err
	}
	cc.vfs.Lock()
	oldConfig := cc.vfs.vfs
	cc.vfs.vfs = vfs
	cc.vfs.Unlock()
	// Need to figure out changed entries and the replay these changes in operationAdd or operationDeleteEntry
	toDelete := diffMap(oldConfig, cc.vfs.vfs)
	toAdd := diffMap(cc.vfs.vfs, oldConfig)
	// Informing Service controller to delete, deleted entries
	for k, vf := range toDelete {
		cc.configCh <- configMessage{op: operationDeleteEntry, pciAddr: k, vf: *vf}
	}
	// Informing Service controller to add, add entries
	for k, vf := range toAdd {
		cc.configCh <- configMessage{op: operationAdd, pciAddr: k, vf: *vf}
	}

	return nil
}

// processConfigMapDelete called when the configmap gets deleted, as a result service controller
// will send a signal to all child processes to exit.
func processConfigMapDelete(cc *configController, configMap *v1.ConfigMap) error {
	logrus.Infof("Processing Delete configmap %s/%s", configMap.ObjectMeta.Namespace, configMap.ObjectMeta.Name)
	// Drop all info from removed config map
	cc.vfs.vfs = map[string]*VF{}
	cc.configCh <- configMessage{op: operationDeleteAll, pciAddr: "", vf: VF{}}

	return nil
}

func buildClient() (*kubernetes.Clientset, error) {
	k8sClientConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}
	k8sClientset, err := kubernetes.NewForConfig(k8sClientConfig)
	if err != nil {
		return nil, err
	}
	return k8sClientset, nil
}

func main() {
	if err := flag.Set("logtostderr", "true"); err != nil {
		logrus.Fatalln(err)
	}
	flag.Parse()

	// creating directory to store pods' network services configuration files
	if _, err := os.Stat(containersConfigPath); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(containersConfigPath, 0644)
		}

		// if os.IsNotExist was true and os.MkdirAll is successful, then err will be nil
		if err != nil {
			logrus.Fatalf("failure to access folder %s with error: %+v", containersConfigPath, err)
		}
	}
	// Instantiating config controller
	cc := newConfigController()
	cc.stopCh = make(chan struct{})
	configCh := make(chan configMessage)
	cc.configCh = configCh
	k8sClientset, err := buildClient()
	if err != nil {
		logrus.Errorf("Failed to build kubernetes client set with error: %+v ", err)
		os.Exit(1)
	}
	cc.k8sClientset = k8sClientset
	// Need to figure out host name since controller has to watch the configmap with VFs which belongs
	// to the local node.
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		logrus.Error("environment variable NODE_NAME must be set via Downward API for the sriov controller, exiting... ")
		os.Exit(1)
	}
	cc.nodeName = nodeName
	// Instantiating service controller
	sc := newServiceController()
	sc.configCh = configCh
	sc.stopCh = make(chan struct{})
	wg.Add(1)
	// Call configController further configuraion and start
	go initConfigController(cc)
	go sc.Run()

	wg.Wait()
}
