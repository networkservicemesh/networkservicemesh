package nsmd

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/networkservice"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	dataplaneapi "github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
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
}

func NewNetworkServiceServer(model model.Model, workspace *Workspace) networkservice.NetworkServiceServer {
	return &networkServiceServer{
		model:     model,
		workspace: workspace,
	}
}

func (srv *networkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logrus.Infof("Received request from client to connect to NetworkService: %#v", request)
	err := ValidateNetworkServiceRequest(request)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	connectionID := srv.model.ConnectionId()
	nscConnection := request.GetConnection()
	// TODO: Mechanism selection
	nscConnection.LocalMechanism = request.LocalMechanismPreference[0]
	_, ok := nscConnection.LocalMechanism.Parameters[LocalMechanismParameterInterfaceNameKey]
	if !ok {
		nscConnection.LocalMechanism.Parameters[LocalMechanismParameterInterfaceNameKey] = nscConnection.GetNetworkService() + connectionID
	}
	netsvc := request.Connection.NetworkService
	if strings.TrimSpace(netsvc) == "" {
		return nil, errors.New("No network service defined")
	}

	endpoints := srv.model.GetNetworkServiceEndpoints(netsvc)

	if len(endpoints) == 0 {
		return nil, errors.New(fmt.Sprintf("netwwork service '%s' not found", request.Connection.NetworkService))
	}

	idx := rand.Intn(len(endpoints))
	endpoint := endpoints[idx]
	if endpoint == nil {
		return nil, errors.New("should not see this error, scaffolding called")
	}

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
	dataplaneClient := dataplaneapi.NewDataplaneOperationsClient(dataplaneConn)

	dpCtx, dpCancel := context.WithTimeout(context.Background(), nseConnectionTimeout)
	defer dpCancel()

	var dpApiConnection *dataplaneapi.Connection
	// If NSE is local, build parameters
	if srv.model.GetNsmUrl() == endpoint.Labels[KEY_NSM_URL] {
		workspace := WorkSpaceRegistry().WorkspaceByEndpoint(endpoint)
		if workspace == nil {
			err := fmt.Errorf("cannot find workspace for endpoint %#v", endpoint)
			logrus.Error(err)
			return nil, err
		}
		nseConn, err := tools.SocketOperationCheck(workspace.NsmClientSocket())
		if err != nil {
			logrus.Errorf("unable to connect to nse %#v", endpoint)
			return nil, err
		}
		defer nseConn.Close()

		client := networkservice.NewNetworkServiceClient(nseConn)
		message := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				ConnectionId:   connectionID,
				NetworkService: endpoint.GetNetworkServiceName(),
				LocalMechanism: &common.LocalMechanism{
					Type:       common.LocalMechanismType_KERNEL_INTERFACE,
					Parameters: map[string]string{},
				},
				ConnectionContext: nscConnection.GetConnectionContext(),
				Labels:            nil,
			},
		}
		nseConnection, e := client.Request(ctx, message)
		err = ValidateConnection(nseConnection, true)
		if err != nil {
			err = fmt.Errorf("failure Validating NSE Connection: %s", err)
			return nil, err
		}

		if e != nil {
			logrus.Errorf("error requesting networkservice from %+v with message %#v error: %s", endpoint, message, e)
			return nil, e
		}

		dpApiConnection = &dataplaneapi.Connection{
			ConnectionContext: nseConnection.GetConnectionContext(),
			ConnectionId:      connectionID,
			LocalSource:       nscConnection.GetLocalMechanism(),
			Destination: &dataplaneapi.Connection_Local{
				Local: nseConnection.GetLocalMechanism(),
			},
		}
	} else {
		// TODO connection is remote, send to nsm
	}
	logrus.Infof("Sending request to dataplane: %#v", dpApiConnection)
	_, err = dataplaneClient.ConnectRequest(dpCtx, dpApiConnection)
	if err != nil {
		logrus.Errorf("Dataplane request failed: %s", err)
		return nil, err
	}

	return &networkservice.Connection{
		ConnectionId:      connectionID,
		NetworkService:    netsvc,
		LocalMechanism:    request.LocalMechanismPreference[0],
		ConnectionContext: dpApiConnection.ConnectionContext,
		Labels:            nil,
	}, nil
}

func (srv *networkServiceServer) Close(context.Context, *networkservice.Connection) (*networkservice.Connection, error) {
	panic("implement me")
}

func (srv *networkServiceServer) Monitor(*networkservice.Connection, networkservice.NetworkService_MonitorServer) error {
	panic("implement me")
}

func (srv *networkServiceServer) MonitorConnections(*common.Empty, networkservice.NetworkService_MonitorConnectionsServer) error {
	panic("implement me")
}
