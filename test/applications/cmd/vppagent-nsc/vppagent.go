package main

import (
	"context"
	"time"

	"github.com/ligato/vpp-agent/api/configurator"

	"github.com/networkservicemesh/networkservicemesh/sdk/compat"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/forwarder/vppagent/pkg/converter"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

func CreateVppInterface(nscConnection *connection.Connection, baseDir string, vppAgentEndpoint string) error {
	conn, err := tools.DialTCPInsecure(vppAgentEndpoint)
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Error(err)
		}
	}()
	client := configurator.NewConfiguratorClient(conn)

	conversionParameters := &converter.ConnectionConversionParameters{
		Name:      "SRC-" + nscConnection.GetId(),
		Terminate: true,
		Side:      converter.SOURCE,
		BaseDir:   baseDir,
	}
	dataChange, err := converter.NewMemifInterfaceConverter(compat.ConnectionUnifiedToLocal(nscConnection), conversionParameters).ToDataRequest(nil, true)

	if err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Infof("Sending DataChange to vppagent: %v", dataChange)
	if _, err := client.Update(context.Background(), &configurator.UpdateRequest{Update: dataChange}); err != nil {
		logrus.Error(err)
		if _, deleteErr := client.Delete(context.Background(), &configurator.DeleteRequest{Delete: dataChange}); deleteErr != nil {
			logrus.Error("unable to delete request", deleteErr)
		}
		return err
	}
	return nil
}

func Reset(vppAgentEndpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if waitErr := tools.WaitForPortAvailable(ctx, "tcp", vppAgentEndpoint, 100*time.Millisecond); waitErr != nil {
		logrus.Error("wait for por available failed", waitErr)
		return waitErr
	}

	conn, err := tools.DialTCPInsecure(vppAgentEndpoint)
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer func() {
		if closeConn := conn.Close(); closeConn != nil {
			logrus.Error(closeConn)
		}
	}()

	client := configurator.NewConfiguratorClient(conn)
	logrus.Infof("Resetting vppagent...")
	_, err = client.Update(context.Background(), &configurator.UpdateRequest{
		Update:     &configurator.Config{},
		FullResync: true,
	})
	if err != nil {
		logrus.Errorf("failed to reset vppagent: %s", err)
	}
	logrus.Infof("Finished resetting vppagent...")
	return nil
}
