// +build unit_test

package nsmmonitor

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/tests"
)

type nsmHelper struct {
	EmptyNSMMonitorHandler
	response  *nsmdapi.ClientConnectionReply
	connected chan bool
	healing   chan bool
}

func (h *nsmHelper) Connected(map[string]*connection.Connection) {
	h.connected <- true
}

func (h *nsmHelper) Healing(conn *connection.Connection) {
	h.healing <- true
}

func TestNSMMonitorInit(t *testing.T) {
	_ = os.Setenv(tools.InsecureEnv, "true")
	g := NewWithT(t)

	storage := tests.NewSharedStorage()
	srv := tests.NewNSMDFullServer(tests.Master, storage)
	defer srv.Stop()
	srv.AddFakeForwarder("test_data_plane", "tcp:some_addr")

	srv.TestModel.AddEndpoint(context.Background(), srv.RegisterFakeEndpoint("golden_network", "test", tests.Master))

	response := srv.RequestNSM("nsm")

	monitorApp := NewNSMMonitorApp(&common.NSConfiguration{
		NsmClientSocket: response.HostBasedir + "/" + response.Workspace + "/" + response.NsmClientSocket,
		NsmServerSocket: response.HostBasedir + "/" + response.Workspace + "/" + response.NsmServerSocket,
		Workspace:       response.HostBasedir + "/" + response.Workspace,
	})
	// Now we could try to connect via Client API
	nsmClient, conn := srv.CreateNSClient(response)
	defer func() {
		err := conn.Close()
		if err != nil {
			logrus.Error(err.Error())
		}
	}()
	request := tests.CreateRequest()

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// Now we need to start monitor and check if it will be able to restore connections.

	connected := make(chan bool)
	healing := make(chan bool)

	monitorApp.SetHandler(&nsmHelper{
		response:  response,
		connected: connected,
		healing:   healing,
	})
	monitorApp.Run()

	select {
	case <-connected:
		logrus.Infof("connected. all fine")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for monitor client to connect")
	}

	logrus.Print("End of test")
}
