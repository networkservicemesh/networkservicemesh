package nsmd

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/networkservice"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	dataplaneapi "github.com/ligato/networkservicemesh/pkg/nsm/apis/dataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type networkServiceServer struct {
	model           model.Model
	nsmPodIPAddress string
}

func NewNetworkServiceServer(model model.Model) networkservice.NetworkServiceServer {
	return &networkServiceServer{
		model: model,
	}
}

func (srv *networkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
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

	connectionID := request.Connection.ConnectionId
	if strings.TrimSpace(connectionID) == "" {
		connectionID = netsvc
	}
	connectionID = request.Connection.ConnectionId + "-" + strconv.FormatUint(rand.Uint64(), 36)

	// get dataplane
	dp, err := srv.model.SelectDataplane()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Programming dataplane: %v...", dp)

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
	if srv.nsmPodIPAddress == endpoint.Labels["nsmurl"] {
		nseConn, err := tools.SocketOperationCheck(endpoint.SocketLocation)
		if err != nil {
			return nil, err
		}
		defer nseConn.Close()

		client := networkservice.NewNetworkServiceClient(nseConn)
		nseConnection, e := client.Request(ctx, &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				ConnectionId:   connectionID,
				NetworkService: "",
				LocalMechanism: &common.LocalMechanism{
					Type:       common.LocalMechanismType_KERNEL_INTERFACE,
					Parameters: map[string]string{},
				},
				ConnectionContext: nil,
				Labels:            nil,
			},
		})

		srcip := strings.Split(nseConnection.LocalMechanism.Parameters["src_ip"], "/")
		if len(srcip) != 2 {
			return nil, errors.New("src_ip is not specified as cidr")
		}
		dstip := strings.Split(nseConnection.LocalMechanism.Parameters["dst_ip"], "/")
		if len(dstip) != 2 {
			return nil, errors.New("dst_ip is not specified as cidr")
		}
		if e != nil {
			return nil, e
		}

		clientMechanism := &common.LocalMechanism{
			Type: common.LocalMechanismType_KERNEL_INTERFACE,
			Parameters: map[string]string{
				nsmutils.NSMkeyNamespace:        nseConnection.LocalMechanism.Parameters["netns"],
				nsmutils.NSMkeyIPv4:             srcip[0],
				nsmutils.NSMkeyIPv4PrefixLength: srcip[1],
			},
		}
		remoteMechanism := &common.LocalMechanism{
			Type: common.LocalMechanismType_KERNEL_INTERFACE,
			Parameters: map[string]string{
				nsmutils.NSMkeyNamespace:        nseConnection.LocalMechanism.Parameters["netns"],
				nsmutils.NSMkeyIPv4:             dstip[0],
				nsmutils.NSMkeyIPv4PrefixLength: dstip[1],
			},
		}
		dpApiConnection = &dataplaneapi.Connection{
			ConnectionId: connectionID,
			LocalSource:  clientMechanism,
			Destination: &dataplaneapi.Connection_Local{
				Local: remoteMechanism,
			},
		}
	} else {
		// TODO connection is remote, send to nsm
	}
	_, err = dataplaneClient.ConnectRequest(dpCtx, dpApiConnection)

	return &networkservice.Connection{
		ConnectionId:      connectionID,
		NetworkService:    netsvc,
		LocalMechanism:    request.LocalMechanismPreference[0],
		ConnectionContext: request.Connection.ConnectionContext,
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
