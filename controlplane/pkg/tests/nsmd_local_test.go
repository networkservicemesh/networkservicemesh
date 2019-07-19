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

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
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

func createRequest(add_exclude bool) *networkservice.NetworkServiceRequest {
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
	if add_exclude {
		request.Connection.GetContext().GetIpContext().ExcludedPrefixes = append(request.Connection.GetContext().GetIpContext().GetExcludedPrefixes(), "127.0.0.1")
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

	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	nsmClient, conn := srv.requestNSMConnection("nsm")
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
		dstIp: "10.20.1.2/30",
	}
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	nsmClient, conn := srv.requestNSMConnection("nsm")
	defer conn.Close()

	request := createRequest(false)

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	println(err.Error())
	Expect(strings.Contains(err.Error(), "failure Validating NSE Connection: ConnectionContext.SrcIp is required cannot be empty/nil")).To(Equal(true))
	Expect(nsmResponse).To(BeNil())
}

//func TestNSEExcludePrefixes(t *testing.T) {
//	RegisterTestingT(t)
//
//	storage := newSharedStorage()
//	srv := newNSMDFullServer(Master, storage, newClusterConfiguration("127.0.0.0/24", "127.0.1.0/24"))
//	defer srv.Stop()
//	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
//	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))
//
//	nsmClient, conn := srv.requestNSMConnection("nsm")
//	defer conn.Close()
//
//	request := createRequest(true)
//
//	nsmResponse, err := nsmClient.Request(context.Background(), request)
//	Expect(err).To(BeNil())
//	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
//	logrus.Print("End of test")
//
//	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
//	Expect(ok).To(Equal(true))
//	Expect(originl.req.Connection.GetContext().GetIpContext().GetExcludedPrefixes()).To(Equal([]string{"127.0.0.1", "127.0.0.0/24", "127.0.1.0/24"}))
//}
//
//func TestNSEExcludePrefixes2(t *testing.T) {
//	RegisterTestingT(t)
//
//	storage := newSharedStorage()
//	srv := newNSMDFullServer(Master, storage, newClusterConfiguration("127.0.0.0/24", "127.0.1.0/24"))
//	defer srv.Stop()
//	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
//	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))
//
//	nsmClient, conn := srv.requestNSMConnection("nsm")
//	defer conn.Close()
//
//	request := createRequest(false)
//
//	nsmResponse, err := nsmClient.Request(context.Background(), request)
//	Expect(err).To(BeNil())
//	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
//	logrus.Print("End of test")
//
//	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
//	Expect(ok).To(Equal(true))
//	Expect(originl.req.Connection.GetContext().GetIpContext().GetExcludedPrefixes()).To(Equal([]string{"127.0.0.0/24", "127.0.1.0/24"}))
//}
//
//func TestExcludePrefixesMonitor(t *testing.T) {
//	RegisterTestingT(t)
//
//	storage := newSharedStorage()
//	srv := newNSMDFullServer(Master, storage, nil)
//	defer srv.Stop()
//
//	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
//	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))
//	ds := srv.serviceRegistry.nseRegistry.getNextSubnetStream()
//
//	ds.addResponse(&registry.SubnetExtendingResponse{
//		Type:   registry.SubnetExtendingResponse_POD,
//		Subnet: "10.32.1.0/24",
//	})
//	ds.addResponse(&registry.SubnetExtendingResponse{
//		Type:   registry.SubnetExtendingResponse_SERVICE,
//		Subnet: "10.96.0.0/12",
//	})
//
//	checkPrefixes(srv, []string{"10.32.1.0/24", "10.96.0.0/12"})
//
//	ds.addResponse(&registry.SubnetExtendingResponse{
//		Type:   registry.SubnetExtendingResponse_POD,
//		Subnet: "10.32.1.0/22",
//	})
//	ds.addResponse(&registry.SubnetExtendingResponse{
//		Type:   registry.SubnetExtendingResponse_SERVICE,
//		Subnet: "10.96.0.0/10",
//	})
//
//	checkPrefixes(srv, []string{"10.32.1.0/22", "10.96.0.0/10"})
//}
//
//func waitForExcludePrefixes(srv *nsmdFullServerImpl, expected []string, timeout time.Duration) bool {
//
//	st := time.Now()
//
//	for ; ; <-time.After(50 * time.Millisecond) {
//		if time.Since(st) > timeout {
//			return false
//		}
//
//		actual := srv.manager.GetExcludePrefixes().GetPrefixes()
//		if len(actual) != len(expected) {
//			continue
//		}
//
//		equal := true
//		for i, e := range expected {
//			if e != actual[i] {
//				equal = false
//				break
//			}
//		}
//
//		if equal {
//			return true
//		}
//	}
//}
//
//func checkPrefixes(srv *nsmdFullServerImpl, expected []string) {
//	success := waitForExcludePrefixes(srv, expected, 5*time.Second)
//	Expect(success).To(BeTrue())
//
//	nsmClient, conn := srv.requestNSMConnection("nsm")
//	defer func() { _ = conn.Close() }()
//
//	request := createRequest(false)
//	nsmResponse, err := nsmClient.Request(context.Background(), request)
//	Expect(err).To(BeNil())
//	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
//
//	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
//	Expect(ok).To(Equal(true))
//	Expect(originl.req.Connection.GetContext().GetIpContext().GetExcludedPrefixes()).To(Equal(expected))
//}
//
//func TestExcludePrefixesMonitorFails(t *testing.T) {
//	RegisterTestingT(t)
//
//	storage := newSharedStorage()
//	srv := newNSMDFullServer(Master, storage, nil)
//	defer srv.Stop()
//
//	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
//	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))
//	ds := srv.serviceRegistry.nseRegistry.getNextSubnetStream()
//
//	ds.addResponse(&registry.SubnetExtendingResponse{
//		Type:   registry.SubnetExtendingResponse_POD,
//		Subnet: "10.32.1.0/24",
//	})
//	ds.addResponse(&registry.SubnetExtendingResponse{
//		Type:   registry.SubnetExtendingResponse_SERVICE,
//		Subnet: "10.96.0.0/12",
//	})
//
//	checkPrefixes(srv, []string{"10.32.1.0/24", "10.96.0.0/12"})
//
//	ds.dummyKill()
//
//	checkPrefixes(srv, []string{"10.32.1.0/24", "10.96.0.0/12"})
//
//	newDs := srv.serviceRegistry.nseRegistry.getNextSubnetStream()
//
//	newDs.addResponse(&registry.SubnetExtendingResponse{
//		Type:   registry.SubnetExtendingResponse_POD,
//		Subnet: "10.32.1.0/22",
//	})
//	newDs.addResponse(&registry.SubnetExtendingResponse{
//		Type:   registry.SubnetExtendingResponse_SERVICE,
//		Subnet: "10.96.0.0/10",
//	})
//
//	checkPrefixes(srv, []string{"10.32.1.0/22", "10.96.0.0/10"})
//}

func TestNSEIPNeghtbours(t *testing.T) {
	RegisterTestingT(t)

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

	request := createRequest(false)

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	logrus.Print("End of test")

	originl, ok := srv.serviceRegistry.localTestNSE.(*nseWithOptions)
	Expect(ok).To(Equal(true))

	Expect(len(originl.connection.GetContext().GetIpContext().GetIpNeighbors())).To(Equal(1))
	Expect(originl.connection.GetContext().GetIpContext().GetIpNeighbors()[0].Ip).To(Equal("127.0.0.1"))
	Expect(originl.connection.GetContext().GetIpContext().GetIpNeighbors()[0].HardwareAddress).To(Equal("ff-ee-ff-ee-ff"))
}

func TestSlowNSE(t *testing.T) {
	RegisterTestingT(t)

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

	request := createRequest(false)

	request.Connection.Labels = map[string]string{}
	request.Connection.Labels["nse_sleep"] = "1"

	ctx, canceOp := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer canceOp()
	nsmResponse, err := nsmClient.Request(ctx, request)
	<-time.After(1 * time.Second)
	println(err.Error())
	Expect(strings.Contains(err.Error(), "rpc error: code = DeadlineExceeded desc = context deadline exceeded")).To(Equal(true))
	Expect(nsmResponse).To(BeNil())
}

func TestSlowDP(t *testing.T) {
	RegisterTestingT(t)

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

	request := createRequest(false)

	request.Connection.Labels = map[string]string{}
	request.Connection.Labels["dataplane_sleep"] = "1"

	ctx, cancelOp := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancelOp()
	nsmResponse, err := nsmClient.Request(ctx, request)
	<-time.After(1 * time.Second)
	println(err.Error())
	Expect(strings.Contains(err.Error(), "rpc error: code = DeadlineExceeded desc = context deadline exceeded")).To(Equal(true))
	Expect(nsmResponse).To(BeNil())
}
