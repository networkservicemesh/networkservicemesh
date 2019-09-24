package serviceregistryserver

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
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
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	conn, err := grpc.DialContext(ctx, m.nsm.Url, grpc.WithInsecure())
	if err != nil {
		return err
	}

	go m.monitorNSM(conn, ctx)

	return nil
}

func (m *nsmMonitor) monitorNSM(conn *grpc.ClientConn, ctx context.Context) {
	defer conn.Close()

	logrus.Infof("NSM Monitor started: %v", m.nsm)

	select {
	case <-ctx.Done():
		m.deleteNSM()
		<-m.nsmDeleted
	case <-m.nsmDeleted:
	}

	logrus.Infof("NSM Monitor done: %v", m.nsm)
}