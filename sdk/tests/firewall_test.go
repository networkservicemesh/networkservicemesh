package tests

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/utils"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent"
)

func TestFirewallMemif(t *testing.T) {
	g := gomega.NewWithT(t)

	rootDir, err := ioutil.TempDir("", "nsmd_test")
	g.Expect(err).To(gomega.BeNil())
	defer utils.ClearFolder(rootDir, false)

	configuration := &common.NSConfiguration{
		MechanismType:    "mem",
		NsmServerSocket:  rootDir + "/server.sock",
		NsmClientSocket:  rootDir + "/client.sock",
		Workspace:        rootDir,
		AdvertiseNseName: "my_network_sercice",
	}

	mechanism, err := connection.NewMechanism(common.MechanismFromString("mem"), "memif_outgoing", "")
	g.Expect(err).To(gomega.BeNil())

	outgoingConnection := &connection.Connection{
		Mechanism:      mechanism,
		NetworkService: "my_network_sercice",
		Id:             "2",
		Context: &connectioncontext.ConnectionContext{
			IpContext: &connectioncontext.IPContext{
				SrcIpRequired: true,
				DstIpRequired: true,
				SrcIpAddr:     "192.168.1.1",
				DstIpAddr:     "192.168.1.2",
			},
		},
	}

	d1 := NewConnectionDump()
	d2 := NewConnectionDump()
	commit := NewTestCommit()

	composite := endpoint.NewCompositeEndpoint(
		endpoint.NewMonitorEndpoint(configuration),
		endpoint.NewConnectionEndpoint(configuration),
		d1,
		NewTestClientEndpoint(outgoingConnection),
		vppagent.NewClientMemifConnect(configuration),
		vppagent.NewMemifConnect(configuration),
		vppagent.NewXConnect(configuration),
		vppagent.NewACL(configuration, map[string]string{}),
		commit,
		d2,
	)

	inMechanism, err := connection.NewMechanism(common.MechanismFromString("mem"), "memif_incoming", "")
	g.Expect(err).To(gomega.BeNil())

	req := networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id:             "1",
			NetworkService: "my_network_sercice",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					SrcIpRequired: true,
					DstIpRequired: true,
				},
			},
		},
		MechanismPreferences: []*connection.Mechanism{inMechanism},
	}
	conn, err := composite.Request(context.Background(), &req)
	g.Expect(conn).ToNot(gomega.BeNil())
	g.Expect(err).To(gomega.BeNil())

	g.Expect(conn.Context.IpContext.SrcIpAddr).To(gomega.Equal("192.168.1.1"))
	g.Expect(conn.Context.IpContext.DstIpAddr).To(gomega.Equal("192.168.1.2"))

	g.Expect(len(commit.VppConfig.VppConfig.Interfaces)).To(gomega.Equal(2))

	g.Expect(commit.VppConfig.VppConfig.Interfaces[0].Name).To(gomega.Equal("2"))
	g.Expect(commit.VppConfig.VppConfig.Interfaces[0].IpAddresses).To(gomega.BeNil())

	g.Expect(commit.VppConfig.VppConfig.Interfaces[1].Name).To(gomega.Equal("1"))
	g.Expect(commit.VppConfig.VppConfig.Interfaces[1].IpAddresses).To(gomega.Equal([]string{"192.168.1.2"}))

	g.Expect(len(commit.VppConfig.VppConfig.XconnectPairs)).To(gomega.Equal(2))
	g.Expect(commit.VppConfig.VppConfig.Interfaces[1].IpAddresses).To(gomega.Equal([]string{"192.168.1.2"}))

	g.Expect(len(d2.ConnectionMap)).To(gomega.Equal(1))

	_, err = composite.Close(context.Background(), conn)
	g.Expect(err).To(gomega.BeNil())

}
