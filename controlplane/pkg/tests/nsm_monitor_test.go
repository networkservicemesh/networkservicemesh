package tests

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsmdapi"
	nsm_sidecars "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/sidecars"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

type nsmHelper struct {
	response  *nsmdapi.ClientConnectionReply
	connected chan bool
	healing   chan bool
	stopped   chan bool
}

func (h *nsmHelper) Stopped() {
	h.stopped <- true
}

func (h *nsmHelper) IsEnableJaeger() bool {
	return false
}

func (h *nsmHelper) Connected(map[string]*connection.Connection) {
	h.connected <- true
}

func (h *nsmHelper) Healing(conn *connection.Connection) {
	h.healing <- true
}

func (h *nsmHelper) GetConfiguration() *common.NSConfiguration {
	return &common.NSConfiguration{
		NsmClientSocket: h.response.HostBasedir + "/" + h.response.Workspace + "/" + h.response.NsmClientSocket,
		NsmServerSocket: h.response.HostBasedir + "/" + h.response.Workspace + "/" + h.response.NsmServerSocket,
		Workspace:       h.response.HostBasedir + "/" + h.response.Workspace,
		TracerEnabled:   false,
	}
}

func (*nsmHelper) ProcessHealing(newConn *connection.Connection, e error) {
}

func TestNSMMonitorInit(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage, defaultClusterConfiguration)
	defer srv.Stop()
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	monitorApp := nsm_sidecars.NewNSMMonitorApp()

	response := srv.requestNSM("nsm")
	// Now we could try to connect via Client API
	nsmClient, conn := srv.createNSClient(response)
	defer func() {
		err := conn.Close()
		if err != nil {
			logrus.Error(err.Error())
		}
	}()
	request := createRequest(false)

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// Now we need to start monitor and check if it will be able to restore connections.

	connected := make(chan bool)
	healing := make(chan bool)
	stoped := make(chan bool)

	monitorApp.SetHandler(&nsmHelper{
		response:  response,
		connected: connected,
		healing:   healing,
		stopped:   stoped,
	})
	monitorApp.Run()

	select {
	case <-connected:
		logrus.Infof("connected. all fine")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for monitor client to connect")
	}

	monitorApp.Stop()

	select {
	case <-stoped:
		logrus.Infof("Monitor stopped")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for monitor client to connect")
	}

	logrus.Print("End of test")
}
