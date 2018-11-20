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

package monitor_connection_server

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/sirupsen/logrus"
)

const (
	InitialChanSize = 10
)

type MonitorConnectionServer interface {
	connection.MonitorConnectionServer
	UpdateConnection(con *connection.Connection)
	DeleteConnection(con *connection.Connection)
	GetConnection(crossconnectId string) (*connection.Connection, bool)
	SendConnectionEvent(event *connection.ConnectionEvent)
}

type monitorConnectionServer struct {
	crossConnectEventCh      chan *connection.ConnectionEvent
	newMonitorRecipientCh    chan connection.MonitorConnection_MonitorConnectionsServer
	closedMonitorRecipientCh chan connection.MonitorConnection_MonitorConnectionsServer

	// monitorRecipients and crossConnects should only ever be updated by the monitor() method
	monitorRecipients []connection.MonitorConnection_MonitorConnectionsServer
	crossConnects     map[string]*connection.Connection
}

func NewMonitorConnectionServer() MonitorConnectionServer {
	// TODO provide some validations here for inputs
	rv := &monitorConnectionServer{
		crossConnects:            make(map[string]*connection.Connection),
		crossConnectEventCh:      make(chan *connection.ConnectionEvent, InitialChanSize),
		newMonitorRecipientCh:    make(chan connection.MonitorConnection_MonitorConnectionsServer, InitialChanSize),
		closedMonitorRecipientCh: make(chan connection.MonitorConnection_MonitorConnectionsServer, InitialChanSize),
	}
	go rv.monitorConnections()
	return rv
}

func (m *monitorConnectionServer) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	filter := NewMonitorConnectionFilter(selector, recipient)
	m.newMonitorRecipientCh <- filter
	go func() {
		select {
		case <-filter.Context().Done():
			m.closedMonitorRecipientCh <- filter
		}
	}()
	return nil
}

func (m *monitorConnectionServer) monitorConnections() {
	for {
		select {
		case newRecipient := <-m.newMonitorRecipientCh:
			initialStateTransferEvent := &connection.ConnectionEvent{
				Type:        connection.ConnectionEventType_INITIAL_STATE_TRANSFER,
				Connections: m.crossConnects,
			}
			newRecipient.Send(initialStateTransferEvent)
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
			for _, con := range crossConnectEvent.GetConnections() {
				if crossConnectEvent.GetType() == connection.ConnectionEventType_UPDATE {
					m.crossConnects[con.GetId()] = con
				}
				if crossConnectEvent.GetType() == connection.ConnectionEventType_DELETE {
					delete(m.crossConnects, con.GetId())
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

func (m *monitorConnectionServer) UpdateConnection(con *connection.Connection) {
	m.crossConnectEventCh <- &connection.ConnectionEvent{
		Type: connection.ConnectionEventType_UPDATE,
		Connections: map[string]*connection.Connection{
			con.GetId(): con,
		},
	}
}

func (m *monitorConnectionServer) DeleteConnection(con *connection.Connection) {
	m.crossConnectEventCh <- &connection.ConnectionEvent{
		Type: connection.ConnectionEventType_DELETE,
		Connections: map[string]*connection.Connection{
			con.GetId(): con,
		},
	}
}

func (m *monitorConnectionServer) GetConnection(connectionId string) (*connection.Connection, bool) {
	con, ok := m.crossConnects[connectionId]
	return con, ok
}

func (m *monitorConnectionServer) SendConnectionEvent(event *connection.ConnectionEvent) {
	m.crossConnectEventCh <- event
}
