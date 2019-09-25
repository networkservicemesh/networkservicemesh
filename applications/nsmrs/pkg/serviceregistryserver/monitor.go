package serviceregistryserver

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/pkg/livemonitor"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type nsmMonitor struct {
	nsm *registry.NetworkServiceManager
	cancel context.CancelFunc
	deleteNSM func()
	nsmDeleted chan int
}

type NsmMonitor interface {
	StartMonitor()
}

func NewNSMMonitor(nsm *registry.NetworkServiceManager, deleteNSM func()) *nsmMonitor {
	nsmDeleted := make(chan int)
	return &nsmMonitor{
		nsm: nsm,
		deleteNSM: deleteNSM,
		nsmDeleted: nsmDeleted,
	}
}

func (m *nsmMonitor) StartMonitor() error {
	conn, err := grpc.Dial(m.nsm.Url, grpc.WithInsecure())
	if err != nil {
		return err
	}

	monitorClient, err := livemonitor.NewClient(conn)
	if err != nil {
		conn.Close()
		return err
	}

	go m.monitorNSM(conn, monitorClient)

	return nil
}

func (m *nsmMonitor) monitorNSM(conn *grpc.ClientConn, monitorClient livemonitor.Client) {
	defer conn.Close()
	defer monitorClient.Close()

	logrus.Infof("NSM Monitor started: %v", m.nsm)

	select {
	case err:=<-monitorClient.ErrorChannel():
		logrus.Errorf("Received error from NSM monitor channel: %v", err)
		go m.deleteNSM()
		<-m.nsmDeleted
	case <-m.nsmDeleted:
	}

	logrus.Infof("NSM Monitor done: %v", m.nsm)
}

func (m *nsmMonitor) stop() {
	m.nsmDeleted <- 0
}