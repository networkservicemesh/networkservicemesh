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
	"sync"

	"github.com/sirupsen/logrus"
)

type serviceInstance struct {
	vfs      map[string]*VF
	stopCh   chan struct{}
	configCh chan configMessage
	doneCh   chan struct{}
}

// first key is network service, second key is pci address of VF
type serviceController struct {
	sriovNetServices map[string]serviceInstance
	configCh         chan configMessage
	// to shut down controller
	stopCh chan struct{}
	// locking for the time of changes
	sync.RWMutex
}

func newServiceController() *serviceController {
	sc := map[string]serviceInstance{}
	return &serviceController{
		sriovNetServices: sc}
}

func (s *serviceController) Run() {
	logrus.Infof("Service Controller is ready, waiting for config messages...")
	for {
		select {
		case <-s.stopCh:
			// global shutdown exiting wait loop
			logrus.Infof("Received global shutdown messages, shutting down all service instances...")
			s.Stop()
			return
		case msg := <-s.configCh:
			switch msg.op {
			case operationAdd:
				logrus.Infof("Service Controller: Received config message to add network service: %s pci address: %s", msg.vf.NetworkService, msg.pciAddr)
				s.processAdd(msg)
			case operationDeleteEntry:
				logrus.Infof("Service Controller: Received config message to delete network service: %s pci address: %s", msg.vf.NetworkService, msg.pciAddr)
				s.processDeleteEntry(msg)
			case operationDeleteAll:
				logrus.Info("Service Controller: Received Delete operation, shutting down service controller")
				s.Stop()
			case operationUpdate:
				logrus.Infof("Service Controller: Received config message to update network service: %s", msg.vf.NetworkService)
				s.processUpdate(msg)
			default:
				logrus.Errorf("error, received message with unknown operation %d", msg.op)
			}
		}
	}
}

func (s *serviceController) processAdd(msg configMessage) {
	// Check if there is already an instance of network service
	_, ok := s.sriovNetServices[msg.vf.NetworkService]
	if !ok {
		// Network Service instance is not found, need to instantiate one
		logrus.Infof("Creating new Network Service instance for Network Service: %s", msg.vf.NetworkService)
		vfs := map[string]*VF{}
		vfs[msg.pciAddr] = &msg.vf
		si := serviceInstance{
			vfs:      vfs,
			configCh: make(chan configMessage),
			stopCh:   make(chan struct{}),
			doneCh:   make(chan struct{}),
		}
		s.sriovNetServices[msg.vf.NetworkService] = si
		// Instantiating Service Instance controller
		sic := newServiceInstanceController(si.configCh, si.stopCh, si.doneCh)
		go sic.Run()
	}
	// Network Service instance already exists, just need to add to VFS map and inform about new VF
	s.Lock()
	defer s.Unlock()
	nsi := s.sriovNetServices[msg.vf.NetworkService]
	nsi.vfs[msg.pciAddr] = &msg.vf
	nsi.configCh <- msg
}

func (s *serviceController) processDeleteEntry(msg configMessage) {
	logrus.Infof("Deleting %s for Network Service instance: %s", msg.pciAddr, msg.vf.NetworkService)
	// Check if there is already an instance of network service
	_, ok := s.sriovNetServices[msg.vf.NetworkService]
	if !ok {
		// Network Service instance is not found, it should not happened
		logrus.Infof("Deleting %s device of Network Service instance: %s, it should not happen as the Network Service instance is unknown ",
			msg.pciAddr, msg.vf.NetworkService)
		return
	}
	// Network Service instance already exists, just need to inform about deleted VF
	nsi := s.sriovNetServices[msg.vf.NetworkService]
	nsi.configCh <- msg
	// Deleting deleted VF from the map
	delete(nsi.vfs, msg.pciAddr)
	// Checking if it was not the last VF in the map
	if len(nsi.vfs) == 0 {
		// Last VF was removed from Network Service Instance, no reason to keep it
		// shutting it down
		logrus.Infof("Network Service instance %s has no more live VFs, stop advertising this resource to the kubelet", msg.vf.NetworkService)
		nsi.stopCh <- struct{}{}
		// Waiting for service instance to close
		<-nsi.doneCh
		// Deleting the Network Service key
		delete(s.sriovNetServices, msg.vf.NetworkService)
	}
}

func (s *serviceController) processUpdate(msg configMessage) {
	// Check if there is already an instance of network service
	_, ok := s.sriovNetServices[msg.vf.NetworkService]
	if !ok {
		// Network Service instance is not found
		logrus.Errorf("fatal error as received update message for non-existing network service %s, ignoring it", msg.vf.NetworkService)
		return
	}
	// Network Service instance already exists, just need to inform about new VF
	nsi := s.sriovNetServices[msg.vf.NetworkService]
	nsi.vfs = map[string]*VF{}
	nsi.configCh <- msg
}

func (s *serviceController) Stop() {
	// Inform all network service instances to shut down
	logrus.Infof("Service controller received shutdown message, will shutdown %d ", len(s.sriovNetServices))
	for s, ns := range s.sriovNetServices {
		ns.stopCh <- struct{}{}
		// Waiting for service instance to close
		<-ns.doneCh
		logrus.Infof("Confirmed shut down of %s instance", s)
	}
	// Re-initializing Network Services map
	s.sriovNetServices = map[string]serviceInstance{}
	logrus.Info("Service controller has completed Service Instance cleanup")
}
