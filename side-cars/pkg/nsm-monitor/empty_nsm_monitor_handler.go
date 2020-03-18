package nsmmonitor

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
)

//EmptyNSMMonitorHandler has empty implementation of each method of interface Handler
type EmptyNSMMonitorHandler struct {
}

//Connected occurs when the nsm-monitor connected
func (h *EmptyNSMMonitorHandler) Connected(map[string]*connection.Connection) {}

//Healing occurs when the healing started
func (h *EmptyNSMMonitorHandler) Healing(conn *connection.Connection) {}

//Closed occurs when the connection closed
func (h *EmptyNSMMonitorHandler) Closed(conn *connection.Connection) {}

//ProcessHealing occurs when the restore failed, the error pass as the second parameter
func (h *EmptyNSMMonitorHandler) ProcessHealing(newConn *connection.Connection, e error) {}

//Updated occurs when the connection updated
func (h *EmptyNSMMonitorHandler) Updated(old, new *connection.Connection) {}

//NewNsmEmptyMonitorHandler creates new empty monitor handler
func NewNsmEmptyMonitorHandler() Handler {
	return &EmptyNSMMonitorHandler{}
}

