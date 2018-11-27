package nsmd

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	dataplaneapi "github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type networkServiceServer struct {
	model     model.Model
	workspace *Workspace
	monitor   monitor_connection_server.MonitorConnectionServer
}

func NewNetworkServiceServer(model model.Model, workspace *Workspace) networkservice.NetworkServiceServer {
	return &networkServiceServer{
		model:     model,
		workspace: workspace,
	}
}

func (srv *networkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Received request from client to connect to NetworkService: %v", request)
	// Make sure its a valid request
	err := request.IsValid()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	// Create a ConnectId for the request.GetConnection()
	request.GetConnection().Id = srv.model.ConnectionId()
	// TODO: Mechanism selection
	request.GetConnection().Mechanism = request.MechanismPreferences[0]
	request.GetConnection().GetMechanism().GetParameters()[connection.Workspace] = srv.workspace.Name()

	// get dataplane
	dp, err := srv.model.SelectDataplane()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Preparing to program dataplane: %v...", dp)

	dataplaneConn, err := tools.SocketOperationCheck(dp.SocketLocation)
	if err != nil {
		return nil, err
	}
	defer dataplaneConn.Close()
	dataplaneClient := dataplaneapi.NewDataplaneClient(dataplaneConn)

	dpCtx, dpCancel := context.WithTimeout(context.Background(), nseConnectionTimeout)
	defer dpCancel()

	endpoint, err := srv.model.SelectEndpoint(request.GetConnection().GetNetworkService())
	if err != nil {
		return nil, err
	}

	var dpApiConnection *crossconnect.CrossConnect
	// If NSE is local, build parameters
	if srv.model.GetNsm().GetName() == endpoint.GetNetworkServiceManager().GetName() {
		workspace := WorkSpaceRegistry().WorkspaceByEndpoint(endpoint.GetNetworkserviceEndpoint())
		if workspace == nil {
			err := fmt.Errorf("cannot find workspace for endpoint %v", endpoint)
			logrus.Error(err)
			return nil, err
		}
		nseConn, err := tools.SocketOperationCheck(workspace.NsmClientSocket())
		if err != nil {
			logrus.Errorf("unable to connect to nse %v", endpoint)
			return nil, err
		}
		defer nseConn.Close()

		client := networkservice.NewNetworkServiceClient(nseConn)
		message := &networkservice.NetworkServiceRequest{
			Connection: &connection.Connection{
				// TODO track connection ids
				Id:             srv.model.ConnectionId(),
				NetworkService: endpoint.GetNetworkService().GetName(),
				Context:        request.GetConnection().GetContext(),
				Labels:         nil,
			},
			MechanismPreferences: []*connection.Mechanism{
				&connection.Mechanism{
					Type:       connection.MechanismType_MEM_INTERFACE,
					Parameters: map[string]string{},
				},
				&connection.Mechanism{
					Type:       connection.MechanismType_KERNEL_INTERFACE,
					Parameters: map[string]string{},
				},
			},
		}
		nseConnection, e := client.Request(ctx, message)
		err = nseConnection.IsComplete()
		if err != nil {
			err = fmt.Errorf("NetworkService.Request() failed with error: %s", err)
			logrus.Error(err)
			return nil, err
		}
		nseConnection.GetMechanism().GetParameters()[connection.Workspace] = workspace.Name()
		request.GetConnection().Context = nseConnection.Context
		err = nseConnection.IsComplete()
		if err != nil {
			err = fmt.Errorf("failure Validating NSE Connection: %s", err)
			return nil, err
		}
		err = request.GetConnection().IsComplete()
		if err != nil {
			err = fmt.Errorf("failure Validating NSC Connection: %s", err)
			return nil, err
		}

		if e != nil {
			logrus.Errorf("error requesting networkservice from %+v with message %#v error: %s", endpoint, message, e)
			return nil, e
		}

		dpApiConnection = &crossconnect.CrossConnect{
			Id:      request.GetConnection().GetId(),
			Payload: endpoint.GetNetworkService().GetPayload(),
			Source: &crossconnect.CrossConnect_LocalSource{
				request.GetConnection(),
			},
			Destination: &crossconnect.CrossConnect_LocalDestination{
				nseConnection,
			},
		}
	} else {
		err := fmt.Errorf("Unable to find NSE matching request locally: %v", request)
		logrus.Error(err)
		return nil, err
	}
	logrus.Infof("Sending request to dataplane: %v", dpApiConnection)
	rv, err := dataplaneClient.Request(dpCtx, dpApiConnection)
	if err != nil {
		logrus.Errorf("Dataplane request failed: %s", err)
		return nil, err
	}
	// TODO - be more cautious here about bad return values from Dataplane
	con := rv.GetSource().(*crossconnect.CrossConnect_LocalSource).LocalSource
	srv.workspace.MonitorConnectionServer().UpdateConnection(con)
	return con, nil
}

func (srv *networkServiceServer) Close(context.Context, *connection.Connection) (*empty.Empty, error) {
	panic("implement me")
}
