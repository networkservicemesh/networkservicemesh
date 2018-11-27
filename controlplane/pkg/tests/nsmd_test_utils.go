package tests

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/nsmdapi"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	remote_networkservice "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type nsmdTestServiceDiscovery struct {
	apiRegistry *testApiRegistry

	services   map[string]*registry.NetworkService
	managers   map[string]*registry.NetworkServiceManager
	endpoints  map[string]*registry.NetworkServiceEndpoint
	nsmCounter int
}

func (impl *nsmdTestServiceDiscovery) RegisterNSE(ctx context.Context, in *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	if in.GetNetworkService() != nil {
		impl.services[in.GetNetworkService().GetName()] = in.GetNetworkService()
	}
	if in.GetNetworkServiceManager() != nil {
		in.NetworkServiceManager.Name = in.GetNetworkServiceManager().Url
		impl.nsmCounter++
		impl.managers[in.GetNetworkServiceManager().Name] = in.GetNetworkServiceManager()
	}
	if in.GetNetworkserviceEndpoint() != nil {
		impl.endpoints[in.GetNetworkserviceEndpoint().EndpointName] = in.GetNetworkserviceEndpoint()
	}
	return in, nil
}

func (impl *nsmdTestServiceDiscovery) RemoveNSE(ctx context.Context, in *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func newNSMDTestServiceDiscovery(testApi *testApiRegistry) *nsmdTestServiceDiscovery {
	return &nsmdTestServiceDiscovery{
		services:    make(map[string]*registry.NetworkService),
		endpoints:   make(map[string]*registry.NetworkServiceEndpoint),
		managers:    make(map[string]*registry.NetworkServiceManager),
		apiRegistry: testApi,
		nsmCounter:  0,
	}
}

func (impl *nsmdTestServiceDiscovery) FindNetworkService(ctx context.Context, in *registry.FindNetworkServiceRequest, opts ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	endpoints := []*registry.NetworkServiceEndpoint{}

	managers := map[string]*registry.NetworkServiceManager{}
	for _, ep := range impl.endpoints {
		if ep.NetworkServiceName == in.NetworkServiceName {
			endpoints = append(endpoints, ep)

			mgr := impl.managers[ep.NetworkServiceManagerName]
			if mgr != nil {
				managers[mgr.Name] = mgr
			}
		}
	}

	return &registry.FindNetworkServiceResponse{
		NetworkService:          impl.services[in.NetworkServiceName],
		NetworkServiceEndpoints: endpoints,
		NetworkServiceManagers:  managers,
	}, nil
}

type nsmdTestServiceRegistry struct {
	nseRegistry             *nsmdTestServiceDiscovery
	apiRegistry             *testApiRegistry
	testDataplaneConnection *testDataplaneConnection
}

func (impl *nsmdTestServiceRegistry) WaitForDataplaneAvailable(model model.Model) {
	// Do Nothing.
}

func (impl *nsmdTestServiceRegistry) WorkspaceName(endpoint *registry.NSERegistration) string {
	return ""
}

func (impl *nsmdTestServiceRegistry) RemoteNetworkServiceClient(nsm *registry.NetworkServiceManager) (remote_networkservice.NetworkServiceClient, *grpc.ClientConn, error) {
	err := tools.WaitForPortAvailable(context.Background(), "tcp", nsm.Url, 1*time.Second)
	if err != nil {
		return nil, nil, err
	}

	logrus.Println("Remote Network Service is available, attempting to connect...")
	conn, err := grpc.Dial(nsm.Url, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("Failed to dial Network Service Registry at %s: %s", nsm.Url, err)
		return nil, nil, err
	}
	client := remote_networkservice.NewNetworkServiceClient(conn)
	return client, conn, nil
}

type localTestNSENetworkServiceClient struct {
}

func (impl *localTestNSENetworkServiceClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	netns, _ := tools.GetCurrentNS()
	if netns == "" {
		netns = "12"
	}
	mechanism := &connection.Mechanism{
		Type: connection.MechanismType_KERNEL_INTERFACE,
		Parameters: map[string]string{
			connection.NetNsInodeKey: netns,
			// TODO: Fix this terrible hack using xid for getting a unique interface name
			connection.InterfaceNameKey: "nsm" + in.GetConnection().GetId(),
		},
	}

	connectionContext := make(map[string]string)

	connectionContext["src_ip"] = "169083138/30"
	connectionContext["dst_ip"] = "169083137/30"

	// TODO take into consideration LocalMechnism preferences sent in request

	connection := &connection.Connection{
		Id:             in.GetConnection().GetId(),
		NetworkService: in.GetConnection().GetNetworkService(),
		Mechanism:      mechanism,
		Context:        connectionContext,
	}
	err := connection.IsComplete()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return connection, nil
}

func (impl *localTestNSENetworkServiceClient) Close(ctx context.Context, in *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	//panic("implement me")
	return nil, nil
}

func (impl *nsmdTestServiceRegistry) EndpointConnection(endpoint *registry.NSERegistration) (networkservice.NetworkServiceClient, *grpc.ClientConn, error) {
	return &localTestNSENetworkServiceClient{}, nil, nil
}

type testDataplaneConnection struct {
	connections []*crossconnect.CrossConnect
}

func (impl *testDataplaneConnection) Request(ctx context.Context, in *crossconnect.CrossConnect, opts ...grpc.CallOption) (*crossconnect.CrossConnect, error) {
	impl.connections = append(impl.connections, in)
	return in, nil
}

func (impl *testDataplaneConnection) Close(ctx context.Context, in *crossconnect.CrossConnect, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func (impl *testDataplaneConnection) MonitorMechanisms(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (dataplane.Dataplane_MonitorMechanismsClient, error) {
	return nil, nil
}

func (impl *nsmdTestServiceRegistry) DataplaneConnection(dataplane *model.Dataplane) (dataplane.DataplaneClient, *grpc.ClientConn, error) {
	return impl.testDataplaneConnection, nil, nil
}

func (impl *nsmdTestServiceRegistry) NSMDApiClient() (nsmdapi.NSMDClient, *grpc.ClientConn, error) {
	addr := fmt.Sprintf("%s:%d", "127.0.0.1", impl.apiRegistry.nsmdPort)
	logrus.Infof("Connecting to nsmd on socket: %s...", addr)

	// Wait to be sure it is already initialized
	err := tools.WaitForPortAvailable(context.Background(), "tcp", addr, 1*time.Second)
	if err != nil {
		return nil, nil, err
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("Failed to dial Network Service Registry at %s: %s", addr, err)
		return nil, nil, err
	}

	logrus.Info("Requesting nsmd for client connection...")
	return nsmdapi.NewNSMDClient(conn), conn, nil
}

func (impl *nsmdTestServiceRegistry) GetPublicAPI() string {
	return fmt.Sprintf("%s:%d", "127.0.0.1", impl.apiRegistry.nsmdPublicPort)
}

func (impl *nsmdTestServiceRegistry) NetworkServiceDiscovery() (registry.NetworkServiceDiscoveryClient, error) {
	return impl.nseRegistry, nil
}

func (impl *nsmdTestServiceRegistry) RegistryClient() (registry.NetworkServiceRegistryClient, error) {
	return impl.nseRegistry, nil
}

func (impl *nsmdTestServiceRegistry) Stop() {
}

type testApiRegistry struct {
	nsmdPort       int
	nsmdPublicPort int
}

func (impl *testApiRegistry) NewNSMServerListener() (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(impl.nsmdPort))
	return listener, err
}

func (impl *testApiRegistry) NewPublicListener() (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(impl.nsmdPublicPort))
	return listener, err
}

var apiPortIterator = 5001

func newTestApiRegistry() *testApiRegistry {
	nsmdPort := apiPortIterator
	nsmdPublicPort := apiPortIterator + 1
	apiPortIterator += 2
	return &testApiRegistry{
		nsmdPort:       nsmdPort,
		nsmdPublicPort: nsmdPublicPort,
	}
}

func newNetworkServiceClient(nsmServerSocket string) (networkservice.NetworkServiceClient, *grpc.ClientConn, error) {
	// Wait till we actually have an nsmd to talk to
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := tools.WaitForPortAvailable(ctx, "unix", nsmServerSocket, 100*time.Millisecond)
	if err != nil {
		return nil, nil, err
	}

	conn, err := tools.SocketOperationCheck(nsmServerSocket)

	if err != nil {
		return nil, nil, err
	}
	// Init related activities start here
	nsmConnectionClient := networkservice.NewNetworkServiceClient(conn)
	return nsmConnectionClient, conn, nil
}

type nsmdFullServer interface {
	Stop()
}
type nsmdFullServerImpl struct {
	apiRegistry     *testApiRegistry
	nseRegistry     *nsmdTestServiceDiscovery
	serviceRegistry *nsmdTestServiceRegistry
	testModel       model.Model
}

func (srv *nsmdFullServerImpl) Stop() {
	srv.serviceRegistry.Stop()
}

func newNSMDFullServer() (*nsmdFullServerImpl, error) {
	srv := &nsmdFullServerImpl{}
	srv.apiRegistry = newTestApiRegistry()
	srv.nseRegistry = newNSMDTestServiceDiscovery(srv.apiRegistry)

	srv.serviceRegistry = &nsmdTestServiceRegistry{
		nseRegistry:             srv.nseRegistry,
		apiRegistry:             srv.apiRegistry,
		testDataplaneConnection: &testDataplaneConnection{},
	}

	srv.testModel = model.NewModel()

	// Lets start NSMD NSE registry service
	err := nsmd.StartNSMServer(srv.testModel, srv.serviceRegistry, srv.apiRegistry)
	if err != nil {
		srv.Stop()
		return nil, err
	}

	err = nsmd.StartAPIServer(srv.testModel, srv.apiRegistry, srv.serviceRegistry)
	if err != nil {
		return nil, err
	}

	return srv, nil
}
