package main

import (
	"context"
	"flag"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/security"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

func parseFlags() string {
	address := flag.String("address", "", "address of crossconnect monitor server")
	flag.Parse()

	return *address
}

type proxyMonitor struct {
	address string
}

func (p *proxyMonitor) MonitorCrossConnects(empty *empty.Empty, src crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	logrus.Infof("MonitorCrossConnects called, address - %v", p.address)

	conn, err := security.GetSecurityManager().DialContext(context.Background(), p.address)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer conn.Close()

	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)

	dstCtx, dstCancel := context.WithCancel(src.Context())
	dst, err := monitorClient.MonitorCrossConnects(dstCtx, empty)
	if err != nil {
		logrus.Error(err)
		return err
	}

	for {
		event, err := dst.Recv()
		if err != nil {
			logrus.Error(err)
			return err
		}

		logrus.Info("Receive event: %v", event)
		if err := src.Send(event); err != nil {
			logrus.Error(err)
			return err
		}

		logrus.Info("Send event: %v", event)

		select {
		case <-src.Context().Done():
			logrus.Info("src context is done")
			dstCancel()
			return nil
		case <-dst.Context().Done():
			logrus.Info("dst context is done")
			return nil
		default:
		}
	}
}

func main() {
	logrus.Info("Starting Xcon Monitor Proxy")
	address := parseFlags()

	logrus.Infof("address=%v", address)

	ln, err := net.Listen("tcp", ":6001")
	if err != nil {
		logrus.Error(err)
		return
	}
	defer ln.Close()

	srv := grpc.NewServer()
	crossconnect.RegisterMonitorCrossConnectServer(srv, &proxyMonitor{
		address: address,
	})

	if err := srv.Serve(ln); err != nil {
		logrus.Error(err)
		return
	}
}
