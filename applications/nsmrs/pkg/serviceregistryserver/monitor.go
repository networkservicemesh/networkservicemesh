package serviceregistryserver

import (
	"io"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/pkg/livemonitor"
)

type nsmMonitor struct {
	nsm        *registry.NetworkServiceManager
	deleteNSM  func()
	nsmDeleted chan int
}

// NsmMonitor - NSMgr liveness monitor
type NsmMonitor interface {
	StartMonitor() error
	Stop()
}

// NewNSMMonitor - creates NSMgr liveness monitor with delete NSM handler
func NewNSMMonitor(nsm *registry.NetworkServiceManager, deleteNSM func()) NsmMonitor {
	nsmDeleted := make(chan int)
	return &nsmMonitor{
		nsm:        nsm,
		deleteNSM:  deleteNSM,
		nsmDeleted: nsmDeleted,
	}
}

// StartMonitor - creates NSMgr liveness monitor client connection
func (m *nsmMonitor) StartMonitor() error {
	conn, err := grpc.Dial(m.nsm.Url, grpc.WithInsecure())
	if err != nil {
		return err
	}

	monitorClient, err := livemonitor.NewClient(conn)
	if err != nil {
		closeErr := conn.Close()
		if closeErr != nil {
			logrus.Errorf("Error closing monitor connection to NSMgr: %v", closeErr)
		}
		return err
	}

	go m.monitorNSM(conn, monitorClient)

	return nil
}

func (m *nsmMonitor) monitorNSM(conn io.Closer, monitorClient livemonitor.Client) {
	defer func() {
		err := conn.Close()
		if err != nil {
			logrus.Errorf("Error closing NSMgr connection: %v", err)
		}
	}()
	defer monitorClient.Close()

	logrus.Infof("NSM Monitor started: %v", m.nsm)

	select {
	case err := <-monitorClient.ErrorChannel():
		logrus.Errorf("Received error from NSM monitor channel: %v", err)
		go m.deleteNSM()
		<-m.nsmDeleted
	case <-m.nsmDeleted:
	}

	logrus.Infof("NSM Monitor done: %v", m.nsm)
}

// Stop - Stops NSMgr liveness monitor and close connection
func (m *nsmMonitor) Stop() {
	m.nsmDeleted <- 0
}
