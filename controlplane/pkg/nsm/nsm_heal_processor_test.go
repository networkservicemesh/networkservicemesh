package nsm

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	test_utils "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/tests/utils"
	. "github.com/onsi/gomega"
	net_context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"testing"
	"time"
)

const (
	networkServiceName = "golden_network"

	localNSMName  = "nsm-local"
	remoteNSMName = "nsm-remote"

	dataplane1Name = "dataplane-1"
	dataplane2Name = "dataplane-2"

	nse1Name = "nse-1"
	nse2Name = "nse-2"
)

var m model.Model
var serviceReg serviceRegistryStub
var conMan connectionManagerStub
var nseMan nseManagerStub
var healProcessor nsmHealProcessor

func setUp() {
	m = model.NewModel()
	serviceReg = serviceRegistryStub{
		discoveryClient: &discoveryClientStub{
			response: createFindNetworkServiceResponse(),
		},
	}
	conMan = connectionManagerStub{}
	nseMan = nseManagerStub{
		nseClients: map[string]*nseClientStub{},
		nses:       []*registry.NSERegistration{},
	}

	healProcessor = nsmHealProcessor{
		serviceRegistry: &serviceReg,
		model:           m,
		properties: &nsm.NsmProperties{
			HealEnabled: true,
		},

		conManager: &conMan,
		nseManager: &nseMan,
	}

	m.SetNsm(&registry.NetworkServiceManager{
		Name: localNSMName,
	})
	m.AddDataplane(&model.Dataplane{
		RegisteredName:       dataplane1Name,
		MechanismsConfigured: true,
	})
}

func TestHealDstDown_RemoteClientLocalEndpoint(t *testing.T) {
	RegisterTestingT(t)
	setUp()

	nse1 := createEndpoint(nse1Name, localNSMName)
	m.AddEndpoint(&model.Endpoint{
		Endpoint: nse1,
	})

	xcon := createCrossConnection(true, false, "src", "dst")
	request := createRequest(false)
	connection := createClientConnection("id", xcon, nse1, remoteNSMName, dataplane1Name, request)
	m.AddClientConnection(connection)

	healed := healProcessor.healDstDown("", connection)
	Expect(healed).To(BeFalse())

	test_utils.NewModelVerifier(m).
		EndpointExists(nse1Name, localNSMName).
		ClientConnectionExists("id", "src", "dst", remoteNSMName, nse1Name, dataplane1Name).
		DataplaneExists(dataplane1Name).
		Verify(t)
}

func TestHealDstDown_LocalClientLocalEndpoint(t *testing.T) {
	RegisterTestingT(t)
	setUp()

	nse1 := createEndpoint(nse1Name, localNSMName)

	nse2 := createEndpoint(nse2Name, localNSMName)
	m.AddEndpoint(&model.Endpoint{
		Endpoint: nse2,
	})

	xcon := createCrossConnection(false, false, "src", "dst")
	request := createRequest(false)
	connection := createClientConnection("id", xcon, nse1, localNSMName, dataplane1Name, request)
	m.AddClientConnection(connection)

	serviceReg.discoveryClient.response = createFindNetworkServiceResponse(nse2)

	healed := healProcessor.healDstDown("", cloneClientConnection(connection))
	Expect(healed).To(BeTrue())

	test_utils.NewModelVerifier(m).
		EndpointNotExists(nse1Name).
		EndpointExists(nse2Name, localNSMName).
		ClientConnectionExists("id", "src", "dst", localNSMName, nse2Name, dataplane1Name).
		DataplaneExists(dataplane1Name).
		Verify(t)
}

func TestHealDstDown_LocalClientLocalEndpoint_NoNSEFound(t *testing.T) {
	RegisterTestingT(t)
	setUp()

	nse1 := createEndpoint(nse1Name, localNSMName)
	m.AddEndpoint(&model.Endpoint{
		Endpoint: nse1,
	})

	xcon := createCrossConnection(false, false, "src", "dst")
	request := createRequest(false)
	connection := createClientConnection("id", xcon, nse1, localNSMName, dataplane1Name, request)
	m.AddClientConnection(connection)

	healed := healProcessor.healDstDown("", cloneClientConnection(connection))
	Expect(healed).To(BeFalse())

	test_utils.NewModelVerifier(m).
		EndpointExists(nse1Name, localNSMName).
		ClientConnectionNotExists("id").
		DataplaneExists(dataplane1Name).
		Verify(t)
}

func TestHealDstDown_LocalClientLocalEndpoint_RequestFailed(t *testing.T) {
	RegisterTestingT(t)
	setUp()

	nse1 := createEndpoint(nse1Name, localNSMName)
	m.AddEndpoint(&model.Endpoint{
		Endpoint: nse1,
	})

	xcon := createCrossConnection(false, false, "src", "dst")
	request := createRequest(false)
	connection := createClientConnection("id", xcon, nse1, localNSMName, dataplane1Name, request)
	m.AddClientConnection(connection)

	conMan.requestError = fmt.Errorf("request error")

	healed := healProcessor.healDstDown("", cloneClientConnection(connection))
	Expect(healed).To(BeFalse())

	test_utils.NewModelVerifier(m).
		EndpointExists(nse1Name, localNSMName).
		ClientConnectionNotExists("id").
		DataplaneExists(dataplane1Name).
		Verify(t)
}

func TestHealDstDown_LocalClientRemoteEndpoint(t *testing.T) {
	RegisterTestingT(t)
	setUp()

	nse1 := createEndpoint(nse1Name, remoteNSMName)

	nse2 := createEndpoint(nse2Name, remoteNSMName)
	nseMan.nses = append(nseMan.nses, nse2)

	xcon := createCrossConnection(false, true, "src", "dst")
	request := createRequest(false)
	connection := createClientConnection("id", xcon, nse1, remoteNSMName, dataplane1Name, request)
	m.AddClientConnection(connection)

	serviceReg.discoveryClient.response = createFindNetworkServiceResponse(nse2)
	conMan.nse = nse2

	healed := healProcessor.healDstDown("", cloneClientConnection(connection))
	Expect(healed).To(BeTrue())

	Expect(nseMan.nseClients[nse1Name].cleanedUp).To(BeTrue())
	Expect(nseMan.nseClients[nse2Name]).To(BeNil())

	test_utils.NewModelVerifier(m).
		EndpointNotExists(nse1Name).
		EndpointNotExists(nse2Name).
		ClientConnectionExists("id", "src", "dst", remoteNSMName, nse2Name, dataplane1Name).
		DataplaneExists(dataplane1Name).
		Verify(t)
}

func TestHealDstDown_LocalClientRemoteEndpoint_NoNSEFound(t *testing.T) {
	RegisterTestingT(t)
	setUp()

	nse1 := createEndpoint(nse1Name, remoteNSMName)

	xcon := createCrossConnection(false, true, "src", "dst")
	request := createRequest(false)
	connection := createClientConnection("id", xcon, nse1, remoteNSMName, dataplane1Name, request)
	m.AddClientConnection(connection)

	healed := healProcessor.healDstDown("", cloneClientConnection(connection))
	Expect(healed).To(BeFalse())

	Expect(nseMan.nseClients[nse1Name].cleanedUp).To(BeTrue())

	test_utils.NewModelVerifier(m).
		EndpointNotExists(nse1Name).
		ClientConnectionNotExists("id").
		DataplaneExists(dataplane1Name).
		Verify(t)
}

type discoveryClientStub struct {
	response *registry.FindNetworkServiceResponse
	error    error
}

func (stub *discoveryClientStub) FindNetworkService(ctx net_context.Context, in *registry.FindNetworkServiceRequest, opts ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	Expect(in.GetNetworkServiceName()).To(Equal(networkServiceName))
	return stub.response, stub.error
}

type serviceRegistryStub struct {
	discoveryClient *discoveryClientStub
	error           error

	serviceregistry.ServiceRegistry
}

func (stub *serviceRegistryStub) DiscoveryClient() (registry.NetworkServiceDiscoveryClient, error) {
	return stub.discoveryClient, stub.error
}

func (stub *serviceRegistryStub) WaitForDataplaneAvailable(model model.Model, timeout time.Duration) error {
	return nsmd.NewServiceRegistry().WaitForDataplaneAvailable(model, timeout)
}

type connectionManagerStub struct {
	requestError error
	nse          *registry.NSERegistration

	closeError error
}

func (stub *connectionManagerStub) request(ctx context.Context, request nsm.NSMRequest, existingConnection *model.ClientConnection) (nsm.NSMConnection, error) {
	if stub.requestError != nil {
		return nil, stub.requestError
	}

	nsmConnection := newConnection(request)
	if nsmConnection.GetNetworkService() != existingConnection.GetNetworkService() {
		if stub.nse != nil {
			existingConnection.Endpoint = stub.nse
		} else {
			m.DeleteClientConnection(existingConnection.GetId())
			return nil, fmt.Errorf("no NSE available")
		}
	}

	existingConnection.ConnectionState = model.ClientConnection_Ready
	existingConnection.DataplaneState = model.DataplaneState_Ready
	m.UpdateClientConnection(existingConnection)

	return nsmConnection, nil
}

func (stub *connectionManagerStub) close(ctx context.Context, clientConnection *model.ClientConnection, closeDataplane bool, modelRemove bool) error {
	if stub.closeError != nil {
		return stub.closeError
	}

	clientConnection.ConnectionState = model.ClientConnection_Closing

	_ = m.DeleteEndpoint(clientConnection.Endpoint.GetNetworkserviceEndpoint().GetEndpointName())

	if closeDataplane {
		m.DeleteDataplane(clientConnection.Dataplane.RegisteredName)
		clientConnection.DataplaneState = model.DataplaneState_None
		if modelRemove {
			m.DeleteClientConnection(clientConnection.ConnectionId)
		}
	}

	return nil
}

type nseClientStub struct {
	cleanedUp bool

	nsm.NetworkServiceClient
}

func (stub *nseClientStub) Cleanup() error {
	stub.cleanedUp = true
	return nil
}

type nseManagerStub struct {
	clientError error
	nseClients  map[string]*nseClientStub

	nses []*registry.NSERegistration
}

func (stub *nseManagerStub) getEndpoint(ctx context.Context, requestConnection nsm.NSMConnection, ignore_endpoints map[string]*registry.NSERegistration) (*registry.NSERegistration, error) {
	panic("implement me")
}

func (stub *nseManagerStub) createNSEClient(ctx context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error) {
	if stub.clientError != nil {
		return nil, stub.clientError
	}

	nseClient := &nseClientStub{
		cleanedUp: false,
	}
	stub.nseClients[endpoint.GetNetworkserviceEndpoint().GetEndpointName()] = nseClient

	return nseClient, nil
}

func (stub *nseManagerStub) isLocalEndpoint(endpoint *registry.NSERegistration) bool {
	return m.GetNsm().GetName() == endpoint.GetNetworkserviceEndpoint().GetNetworkServiceManagerName()
}

func (stub *nseManagerStub) checkUpdateNSE(ctx context.Context, reg *registry.NSERegistration) bool {
	for _, nse := range stub.nses {
		if nse.GetNetworkserviceEndpoint().GetEndpointName() == reg.GetNetworkserviceEndpoint().GetEndpointName() {
			return true
		}
	}

	return false
}

func createEndpoint(nse, nsm string) *registry.NSERegistration {
	return &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name: networkServiceName,
		},
		NetworkServiceManager: &registry.NetworkServiceManager{
			Name: nsm,
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName:        networkServiceName,
			NetworkServiceManagerName: nsm,
			EndpointName:              nse,
		},
	}
}

func createCrossConnection(isRemoteSrc, isRemoteDst bool, srcID, dstID string) *crossconnect.CrossConnect {
	xcon := &crossconnect.CrossConnect{}

	if isRemoteSrc {
		xcon.Source = &crossconnect.CrossConnect_RemoteSource{
			RemoteSource: &remote_connection.Connection{
				Id: srcID,
			},
		}
	} else {
		xcon.Source = &crossconnect.CrossConnect_LocalSource{
			LocalSource: &local_connection.Connection{
				Id: srcID,
			},
		}
	}

	if isRemoteDst {
		xcon.Destination = &crossconnect.CrossConnect_RemoteDestination{
			RemoteDestination: &remote_connection.Connection{
				Id: dstID,
			},
		}
	} else {
		xcon.Destination = &crossconnect.CrossConnect_LocalDestination{
			LocalDestination: &local_connection.Connection{
				Id: dstID,
			},
		}
	}

	return xcon
}

func cloneCrossConnection(xcon *crossconnect.CrossConnect) *crossconnect.CrossConnect {
	var isRemoteSrc, isRemoteDst bool
	var srcID, dstID string

	if source := xcon.GetRemoteSource(); source != nil {
		isRemoteSrc = true
		srcID = source.GetId()
	} else if source := xcon.GetLocalSource(); source != nil {
		isRemoteSrc = false
		srcID = source.GetId()
	}

	if destination := xcon.GetRemoteDestination(); destination != nil {
		isRemoteDst = true
		dstID = destination.GetId()
	} else if destination := xcon.GetLocalDestination(); destination != nil {
		isRemoteDst = false
		dstID = destination.GetId()
	}

	return createCrossConnection(isRemoteSrc, isRemoteDst, srcID, dstID)
}

func createRequest(isRemote bool) nsm.NSMRequest {
	if isRemote {
		return &remote_networkservice.NetworkServiceRequest{
			Connection: &remote_connection.Connection{
				NetworkService: networkServiceName,
			},
		}
	} else {
		return &local_networkservice.NetworkServiceRequest{
			Connection: &local_connection.Connection{
				NetworkService: networkServiceName,
			},
		}
	}
}

func createClientConnection(id string, xcon *crossconnect.CrossConnect, nse *registry.NSERegistration, nsm, dataplane string, request nsm.NSMRequest) *model.ClientConnection {
	return &model.ClientConnection{
		ConnectionId: id,
		Xcon:         xcon,
		RemoteNsm: &registry.NetworkServiceManager{
			Name: nsm,
		},
		Endpoint:       nse,
		Dataplane:      m.GetDataplane(dataplane),
		Request:        request,
		DataplaneState: model.DataplaneState_Ready,
	}
}

func cloneClientConnection(connection *model.ClientConnection) *model.ClientConnection {
	id := connection.GetId()
	xcon := cloneCrossConnection(connection.Xcon)
	nse := createEndpoint(connection.Endpoint.GetNetworkserviceEndpoint().GetEndpointName(), connection.Endpoint.GetNetworkServiceManager().GetName())
	nsm := connection.RemoteNsm.GetName()
	dataplane := connection.Dataplane.RegisteredName
	request := createRequest(connection.Request.IsRemote())

	return createClientConnection(id, xcon, nse, nsm, dataplane, request)
}

func createFindNetworkServiceResponse(nses ...*registry.NSERegistration) *registry.FindNetworkServiceResponse {
	response := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name: networkServiceName,
		},
		NetworkServiceManagers:  map[string]*registry.NetworkServiceManager{},
		NetworkServiceEndpoints: []*registry.NetworkServiceEndpoint{},
	}

	for _, nse := range nses {
		nsm := nse.GetNetworkServiceManager().GetName()
		response.NetworkServiceManagers[nsm] = &registry.NetworkServiceManager{
			Name: nsm,
		}
		response.NetworkServiceEndpoints = append(response.NetworkServiceEndpoints, nse.NetworkserviceEndpoint)
	}

	return response
}
