package main

import (
	"context"
	"time"

	"github.com/ligato/vpp-agent/api/configurator"

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
	defer conn.Close()
	client := configurator.NewConfiguratorClient(conn)

	conversionParameters := &converter.ConnectionConversionParameters{
		Name:      "SRC-" + nscConnection.GetId(),
		Terminate: true,
		Side:      converter.SOURCE,
		BaseDir:   baseDir,
	}
	dataChange, err := converter.NewMemifInterfaceConverter(nscConnection, conversionParameters).ToDataRequest(nil, true)

	if err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Infof("Sending DataChange to vppagent: %v", dataChange)
	if _, err := client.Update(context.Background(), &configurator.UpdateRequest{Update: dataChange, FullResync: true}); err != nil {
		logrus.Error(err)
		client.Delete(context.Background(), &configurator.DeleteRequest{Delete: dataChange})
		return err
	}
	return nil
}

func Reset(vppAgentEndpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	tools.WaitForPortAvailable(ctx, "tcp", vppAgentEndpoint, 100*time.Millisecond)

	conn, err := tools.DialTCPInsecure(vppAgentEndpoint)
	if err != nil {
		logrus.Errorf("can't dial grpc server: %v", err)
		return err
	}
	defer conn.Close()

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
