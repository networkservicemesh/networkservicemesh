package main

import (
	"context"
	"flag"
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

const (
	proxyMonitorAddress = "127.0.0.1:6001"
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

	conn, err := tools.DialTCP(p.address)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer func() { _ = conn.Close() }()

	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)

	dstCtx, dstCancel := context.WithCancel(src.Context())
	defer dstCancel()

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

		logrus.Infof("Receive event: %v", event)
		if err := src.Send(event); err != nil {
			logrus.Error(err)
			return err
		}

		logrus.Infof("Send event: %v", event)

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
	utils.PrintAllEnv(logrus.StandardLogger())
	address := parseFlags()

	logrus.Infof("address=%v", address)

	ln, err := net.Listen("tcp", proxyMonitorAddress)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer func() { _ = ln.Close() }()

	srv := grpc.NewServer()
	crossconnect.RegisterMonitorCrossConnectServer(srv, &proxyMonitor{
		address: address,
	})

	if err := srv.Serve(ln); err != nil {
		logrus.Error(err)
		return
	}
}
