package tests

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	context2 "golang.org/x/net/context"
	"google.golang.org/grpc"
	"os"
	"strings"
	"testing"
)

type nseWithOptions struct {
	netns             string
	srcIp             string
	dstIp             string
	needMechanism     bool
	need_ip_neighbors bool
	connection        *connection.Connection
}

func (impl *nseWithOptions) Request(ctx context2.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	var mechanism *connection.Mechanism
	if impl.needMechanism {
		mechanism = &connection.Mechanism{
			Type: connection.MechanismType_KERNEL_INTERFACE,
			Parameters: map[string]string{
				connection.NetNsInodeKey: impl.netns,
				// TODO: Fix this terrible hack using xid for getting a unique interface name
				connection.InterfaceNameKey: "nsm" + in.GetConnection().GetId(),
			},
		}
	}

	conn := &connection.Connection{
		Id:             in.GetConnection().GetId(),
		NetworkService: in.GetConnection().GetNetworkService(),
		Mechanism:      mechanism,
		Context: &connectioncontext.ConnectionContext{
			SrcIpAddr: impl.srcIp,
			DstIpAddr: impl.dstIp,
		},
	}

	if impl.need_ip_neighbors {
		conn.GetContext().IpNeighbors = []*connectioncontext.IpNeighbor{
			&connectioncontext.IpNeighbor{
				Ip:              "127.0.0.1",
				HardwareAddress: "ff-ee-ff-ee-ff",
			},
		}
	}
	impl.connection = conn
	return conn, nil
}

func createRequest(add_exclude bool) *networkservice.NetworkServiceRequest {
	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					connection.NetNsInodeKey:    "10",
					connection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}
	if add_exclude {
		request.Connection.Context.ExcludedPrefixes = append(request.Connection.Context.ExcludedPrefixes, "127.0.0.1")
	}

	return request
}

func (nseWithOptions) Close(ctx context2.Context, in *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

// Below only tests

func TestNSMDRequestClientConnectionRequest(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	srv.registerFakeEndpoint("golden_network", "test", Master)

	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := createRequest(false)

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	logrus.Print("End of test")
}

func TestNSENoSrc(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()

	srv.serviceRegistry.localTestNSE = &nseWithOptions{
		netns: "12",
		//srcIp: "169083138/30",
		dstIp: "169083137/30",
	}
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	srv.registerFakeEndpoint("golden_network", "test", Master)

	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := createRequest(false)

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	println(err.Error())
	Expect(strings.Contains(err.Error(), "failure Validating NSE Connection: ConnectionContext.SrcIp is required cannot be empty/nil")).To(Equal(true))
	Expect(nsmResponse).To(BeNil())
}

func TestNSEExcludePrefixes(t *testing.T) {
	RegisterTestingT(t)

	err := os.Setenv(nsmd.ExcludedPrefixesEnv, "127.0.0.1/24, abc")

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
	srv.registerFakeEndpoint("golden_network", "test", Master)

	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := createRequest(true)

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	logrus.Print("End of test")

	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
	Expect(ok).To(Equal(true))
	Expect(originl.req.Connection.Context.ExcludedPrefixes).To(Equal([]string{"127.0.0.1", "127.0.0.1/24", "abc"}))
}

func TestNSEExcludePrefixes2(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
	srv.registerFakeEndpoint("golden_network", "test", Master)

	err := os.Setenv(nsmd.ExcludedPrefixesEnv, "127.0.0.1/24, abc")

	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := createRequest(false)

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	logrus.Print("End of test")

	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
	Expect(ok).To(Equal(true))
	Expect(originl.req.Connection.Context.ExcludedPrefixes).To(Equal([]string{"127.0.0.1/24", "abc"}))
}

func TestNSEIPNeghtbours(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()
	srv.serviceRegistry.localTestNSE = &nseWithOptions{
		netns:             "12",
		srcIp:             "169083138/30",
		dstIp:             "169083137/30",
		need_ip_neighbors: true,
	}

	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
	srv.registerFakeEndpoint("golden_network", "test", Master)

	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := createRequest(false)

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	logrus.Print("End of test")

	originl, ok := srv.serviceRegistry.localTestNSE.(*nseWithOptions)
	Expect(ok).To(Equal(true))

	Expect(len(originl.connection.Context.IpNeighbors)).To(Equal(1))
	Expect(originl.connection.Context.IpNeighbors[0].Ip).To(Equal("127.0.0.1"))
	Expect(originl.connection.Context.IpNeighbors[0].HardwareAddress).To(Equal("ff-ee-ff-ee-ff"))
}
