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

package monitor_crossconnect_server

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/sirupsen/logrus"
)

type MonitorCrossConnectServer interface {
	crossconnect.MonitorCrossConnectServer
	UpdateCrossConnect(xcon *crossconnect.CrossConnect)
	DeleteCrossConnect(xcon *crossconnect.CrossConnect)
}

type monitorCrossConnectServer struct {
	crossConnectEventCh      chan *crossconnect.CrossConnectEvent
	newMonitorRecipientCh    chan crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer
	closedMonitorRecipientCh chan crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer

	// monitorRecipients and crossConnects should only ever be updated by the monitor() method
	monitorRecipients []crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer
	crossConnects     map[string]*crossconnect.CrossConnect
}

func NewMonitorCrossConnectServer() MonitorCrossConnectServer {
	// TODO provide some validations here for inputs
	rv := &monitorCrossConnectServer{
		crossConnects:            make(map[string]*crossconnect.CrossConnect),
		crossConnectEventCh:      make(chan *crossconnect.CrossConnectEvent, 10),
		newMonitorRecipientCh:    make(chan crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer, 10),
		closedMonitorRecipientCh: make(chan crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer, 10),
	}
	go rv.monitorCrossConnects()
	return rv
}

func (m *monitorCrossConnectServer) MonitorCrossConnects(_ *empty.Empty, recipient crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	m.newMonitorRecipientCh <- recipient

	// We need to wait until it will be done and do not exit
	for {
		select {
		case <-recipient.Context().Done():
			m.closedMonitorRecipientCh <- recipient
			return nil
		}
	}
}

func (m *monitorCrossConnectServer) monitorCrossConnects() {
	for {
		select {
		case newRecipient := <-m.newMonitorRecipientCh:
			initialStateTransferEvent := &crossconnect.CrossConnectEvent{
				Type:          crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
				CrossConnects: m.crossConnects,
			}
			err := newRecipient.Send(initialStateTransferEvent)
			if err != nil {
				logrus.Errorf("Error during send: %+v", err)
			}
			m.monitorRecipients = append(m.monitorRecipients, newRecipient)
			// TODO handle case where a monitorRecipient goes away
		case closedRecipent := <-m.closedMonitorRecipientCh:
			for j, r := range m.monitorRecipients {
				if r == closedRecipent {
					m.monitorRecipients = append(m.monitorRecipients[:j], m.monitorRecipients[j+1:]...)
					break
				}
			}
		case crossConnectEvent := <-m.crossConnectEventCh:
			for _, xcon := range crossConnectEvent.GetCrossConnects() {
				if crossConnectEvent.GetType() == crossconnect.CrossConnectEventType_UPDATE {
					m.crossConnects[xcon.GetId()] = xcon
				}
				if crossConnectEvent.GetType() == crossconnect.CrossConnectEventType_DELETE {
					delete(m.crossConnects, xcon.GetId())
				}
			}
			for _, recipient := range m.monitorRecipients {
				err := recipient.Send(crossConnectEvent)
				if err != nil {
					logrus.Errorf("Received error trying to send crossConnectEvent %v to %v: %s", crossConnectEvent, recipient, err)
				}
			}
		}
	}
}

func (m *monitorCrossConnectServer) UpdateCrossConnect(xcon *crossconnect.CrossConnect) {
	m.crossConnectEventCh <- &crossconnect.CrossConnectEvent{
		Type: crossconnect.CrossConnectEventType_UPDATE,
		CrossConnects: map[string]*crossconnect.CrossConnect{
			xcon.GetId(): xcon,
		},
	}
}

func (m *monitorCrossConnectServer) DeleteCrossConnect(xcon *crossconnect.CrossConnect) {
	m.crossConnectEventCh <- &crossconnect.CrossConnectEvent{
		Type: crossconnect.CrossConnectEventType_DELETE,
		CrossConnects: map[string]*crossconnect.CrossConnect{
			xcon.GetId(): xcon,
		},
	}
}

func (m *monitorCrossConnectServer) GetCrossConnect(crossconnectId string) (*crossconnect.CrossConnect, bool) {
	xcon, ok := m.crossConnects[crossconnectId]
	return xcon, ok
}

func (m *monitorCrossConnectServer) SendCrossConnectEvent(event *crossconnect.CrossConnectEvent) {
	m.crossConnectEventCh <- event
}
