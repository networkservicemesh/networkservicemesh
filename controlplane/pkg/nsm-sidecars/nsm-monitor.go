package nsm_sidecars

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const (
	nscLogFormat          = "NSM Monitor: %v"
	nscLogWithParamFormat = "NSM Monitor: %v: %v"
)

// NSMMonitorHelper - helper to perform configuration of monitoring app required for testing.
type NSMMonitorHelper interface {
	Connected(map[string]*connection.Connection)
	Healing(conn *connection.Connection)
	GetConfiguration() *common.NSConfiguration
	ProcessHealing(newConn *connection.Connection, e error)
	Stopped()
	IsEnableJaeger() bool
}

// NSMMonitorApp - application to perform monitoring.
type NSMMonitorApp interface {
	// Run - run application with printing version
	Run(version string)
	// SetHelper - sets a helper instance.
	SetHelper(helper NSMMonitorHelper)
	Stop()
}

type nsmMonitorApp struct {
	connections map[string]*connection.Connection
	helper      NSMMonitorHelper
	stop        chan bool
}

func (c *nsmMonitorApp) Stop() {
	c.stop <- true
}

// SetHelper - sets a helper class
func (c *nsmMonitorApp) SetHelper(listener NSMMonitorHelper) {
	c.helper = listener
}

func (c *nsmMonitorApp) Run(version string) {
	logrus.Infof(nscLogFormat, "Starting")
	logrus.Infof("Version: %v", version)
	// Capture signals to cleanup before exiting
	if c.helper == nil || c.helper.IsEnableJaeger() {
		tracer, closer := tools.InitJaeger("nsm-monitor")
		opentracing.SetGlobalTracer(tracer)
		defer func() { _ = closer.Close() }()
	}

	go c.beginMonitoring()
}

// NewNSMMonitorApp - creates a monitoring application.
func NewNSMMonitorApp() NSMMonitorApp {
	return &nsmMonitorApp{
		connections: map[string]*connection.Connection{},
		stop: make(chan bool),
	}
}

func (c *nsmMonitorApp) beginMonitoring() {
	for {
		var configuration *common.NSConfiguration
		if c.helper != nil {
			configuration = c.helper.GetConfiguration()
		}
		nsc, err := client.NewNSMClient(context.Background(), configuration)
		if err != nil {
			logrus.Fatalf(nscLogWithParamFormat, "Unable to create the NSM client", err)
		}

		logrus.Info(nscLogFormat, "connection to NSM established")

		monitorClient, err := local.NewMonitorClient(nsc.NsmConnection.GrpcClient)
		if err != nil {
			logrus.Fatalf(nscLogWithParamFormat, "failed to start monitor client", err)
		}
		defer monitorClient.Close()

		initRecieved := false
		recovery := false

		for {
			if initRecieved && !recovery {
				// Performing recovery if required.
				if c.helper != nil {
					c.helper.Connected(c.connections)
				}
				recovery = true
				// Since NSMD will setup public socket only when all connections will be ok, we need to perform request only on ones it loose.
				logrus.Infof(nscLogFormat, "Performing recovery if needed...")

				needRetry := false
				for _, conn := range c.connections {
					if conn.State == connection.State_UP {
						continue
					}
					cClone := (conn.Clone()).(*connection.Connection)

					ipCtx := cClone.Context.IpContext
					if ipCtx != nil {
						if ipCtx.DstIpAddr != "" {
							ipCtx.DstIpRequired = true
						}
						if ipCtx.SrcIpAddr != "" {
							ipCtx.SrcIpRequired = true
						}
					}

					outgoingRequest := networkservice.NetworkServiceRequest{
						Connection: cClone,
						MechanismPreferences: []*connection.Mechanism{
							conn.Mechanism,
						},
					}
					if c.helper != nil {
						c.helper.Healing(cClone)
					}

					newConn, err := nsc.PerformRequest(&outgoingRequest)
					if err != nil {
						logrus.Errorf(nscLogWithParamFormat, "failed to restore connection. Will retry", err)
						// Let's drop connection id, since we failed one time.
						conn.Id = "-"
						needRetry = true
						continue;
					} else {
						logrus.Errorf(nscLogWithParamFormat, "connection restored", newConn)
						delete(c.connections, conn.Id)
						c.connections[newConn.Id] = newConn
					}
					if c.helper != nil {
						c.helper.ProcessHealing(newConn, err)
					}
				}
				if needRetry {
					continue
				}
			}
			select {
			case err = <-monitorClient.ErrorChannel():
				logrus.Fatalf(nscLogWithParamFormat, "NSM die, re-connecting", err)
				for _, c := range c.connections {
					c.State = connection.State_DOWN // Mark all as down.
				}
				continue
			case event := <-monitorClient.EventChannel():
				if event.EventType() == monitor.EventTypeInitialStateTransfer {
					logrus.Infof(nscLogFormat, "Monitor started")
					initRecieved = true
				}

				for _, entity := range event.Entities() {
					switch event.EventType() {
					case monitor.EventTypeInitialStateTransfer, monitor.EventTypeUpdate:
						conn, ok := entity.(*connection.Connection)
						// update connections
						if ok {
							if existingConn, exists := c.connections[conn.Id]; exists {
								logrus.Infof(nscLogWithParamFormat, "Connection updated", fmt.Sprint("%v %v", existingConn, conn))
							} else {
								logrus.Infof(nscLogWithParamFormat, "Initial connection accepted", conn)
							}
							c.connections[conn.Id] = conn
						}
					case monitor.EventTypeDelete:
						logrus.Infof(nscLogFormat, "Connection closed")
						return
					}
				}
			case <-c.stop:
				if c.helper != nil {
					c.helper.Stopped()
					logrus.Infof(nscLogFormat, "Processing stop")
					return
				}
			}
		}
	}
}
