package tests

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	context2 "golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
)

type nseWithOptions struct {
	netns             string
	srcIp             string
	dstIp             string
	need_ip_neighbors bool
	connection        *connection.Connection
}

func (impl *nseWithOptions) Request(ctx context2.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	var mechanism *connection.Mechanism

	if in.Connection.Labels != nil {
		if val, ok := in.Connection.Labels["nse_sleep"]; ok {
			delay, err := strconv.Atoi(val)
			if err == nil {
				logrus.Infof("Delaying NSE init: %v", delay)
				<-time.After(time.Duration(delay) * time.Second)
			}
		}
	}
	mechanism = &connection.Mechanism{
		Type: in.MechanismPreferences[0].Type,
		Parameters: map[string]string{
			connection.NetNsInodeKey: impl.netns,
			// TODO: Fix this terrible hack using xid for getting a unique interface name
			connection.InterfaceNameKey: "nsm" + in.GetConnection().GetId(),
		},
	}

	conn := &connection.Connection{
		Id:             in.GetConnection().GetId(),
		NetworkService: in.GetConnection().GetNetworkService(),
		Mechanism:      mechanism,
		Context: &connectioncontext.ConnectionContext{
			IpContext: &connectioncontext.IPContext{
				SrcIpAddr: impl.srcIp,
				DstIpAddr: impl.dstIp,
			},
		},
	}

	if impl.need_ip_neighbors {
		conn.GetContext().GetIpContext().IpNeighbors = []*connectioncontext.IpNeighbor{
			&connectioncontext.IpNeighbor{
				Ip:              "127.0.0.1",
				HardwareAddress: "ff-ee-ff-ee-ff",
			},
		}
	}
	impl.connection = conn
	return conn, nil
}

func createRequest() *networkservice.NetworkServiceRequest {
	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
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

	return request
}

func (nseWithOptions) Close(ctx context2.Context, in *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

// Below only tests

func TestNSMDRequestClientConnectionRequest(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	nsmClient, conn := srv.requestNSMConnection("nsm")
	defer conn.Close()

	request := createRequest()

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	logrus.Print("End of test")
}

func TestNSENoSrc(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()

	srv.serviceRegistry.localTestNSE = &nseWithOptions{
		netns: "12",
		//srcIp: "169083138/30",
		dstIp: "10.20.1.2/30",
	}
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	nsmClient, conn := srv.requestNSMConnection("nsm")
	defer conn.Close()

	request := createRequest()

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	println(err.Error())
	g.Expect(strings.Contains(err.Error(), "failure Validating NSE Connection: ConnectionContext.SrcIp is required cannot be empty/nil")).To(Equal(true))
	g.Expect(nsmResponse).To(BeNil())
}

func TestNSEIPNeghtbours(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()
	srv.serviceRegistry.localTestNSE = &nseWithOptions{
		netns:             "12",
		srcIp:             "10.20.1.1/30",
		dstIp:             "10.20.1.2/30",
		need_ip_neighbors: true,
	}

	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	nsmClient, conn := srv.requestNSMConnection("nsm")
	defer conn.Close()

	request := createRequest()

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	logrus.Print("End of test")

	originl, ok := srv.serviceRegistry.localTestNSE.(*nseWithOptions)
	g.Expect(ok).To(Equal(true))

	g.Expect(len(originl.connection.GetContext().GetIpContext().GetIpNeighbors())).To(Equal(1))
	g.Expect(originl.connection.GetContext().GetIpContext().GetIpNeighbors()[0].Ip).To(Equal("127.0.0.1"))
	g.Expect(originl.connection.GetContext().GetIpContext().GetIpNeighbors()[0].HardwareAddress).To(Equal("ff-ee-ff-ee-ff"))
}

func TestSlowNSE(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()

	srv.serviceRegistry.localTestNSE = &nseWithOptions{
		netns: "12",
		srcIp: "169083138/30",
		dstIp: "169083137/30",
	}
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	nsmClient, conn := srv.requestNSMConnection("nsm")
	defer conn.Close()

	request := createRequest()

	request.Connection.Labels = map[string]string{}
	request.Connection.Labels["nse_sleep"] = "1"

	ctx, canceOp := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer canceOp()
	nsmResponse, err := nsmClient.Request(ctx, request)
	<-time.After(1 * time.Second)
	println(err.Error())
	g.Expect(strings.Contains(err.Error(), "rpc error: code = DeadlineExceeded desc = context deadline exceeded")).To(Equal(true))
	g.Expect(nsmResponse).To(BeNil())
}

func TestSlowDP(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()

	srv.serviceRegistry.localTestNSE = &nseWithOptions{
		netns: "12",
		srcIp: "10.20.1.1/30",
		dstIp: "10.20.1.2/30",
	}
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	nsmClient, conn := srv.requestNSMConnection("nsm")
	defer conn.Close()

	request := createRequest()

	request.Connection.Labels = map[string]string{}
	request.Connection.Labels["dataplane_sleep"] = "1"

	ctx, cancelOp := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancelOp()
	nsmResponse, err := nsmClient.Request(ctx, request)
	<-time.After(1 * time.Second)
	println(err.Error())
	g.Expect(strings.Contains(err.Error(), "rpc error: code = DeadlineExceeded desc = context deadline exceeded")).To(Equal(true))
	g.Expect(nsmResponse).To(BeNil())
}
