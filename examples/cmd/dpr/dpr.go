package main

import (
	"time"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	remote "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	dataplaneapi "github.com/ligato/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// To check if the tunnel was created use:
// kubectl exec nsm-vppagent-dataplane-<ID> vppctl show vxlan tunnel

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
	nsmDataplenSock      = "/var/lib/networkservicemesh/nsm-vppagent.dataplane.sock"
)

func main() {

	logrus.Infof("Preparing to program dataplane: %v...", "vpp")

	dataplaneConn, err := tools.SocketOperationCheck(nsmDataplenSock)
	if err != nil {
		return
	}
	defer dataplaneConn.Close()
	dataplaneClient := dataplaneapi.NewDataplaneClient(dataplaneConn)

	dpCtx, dpCancel := context.WithTimeout(context.Background(), nseConnectionTimeout)
	defer dpCancel()

	// Create a dummy Remote-Remote request. It does not make much sense in reality.
	// But this is easier than specifying a full Local connection.
	dpAPIConnection := &crossconnect.CrossConnect{
		Id:      "0",
		Payload: "IP",
		Source: &crossconnect.CrossConnect_RemoteSource{
			&remote.Connection{
				Id:             "1",
				NetworkService: "NSE_1",
				Context: map[string]string{
					"Param": "Context1",
				},
				Mechanism: &remote.Mechanism{
					Type: remote.MechanismType_VXLAN,
					Parameters: map[string]string{
						"src_ip": "192.168.111.1",
						"dst_ip": "192.168.111.2",
						"vni":    "127",
					},
				},
			},
		},
		Destination: &crossconnect.CrossConnect_RemoteDestination{
			&remote.Connection{
				Id:             "2",
				NetworkService: "NSE_2",
				Context: map[string]string{
					"Param": "Context2",
				},
				Mechanism: &remote.Mechanism{
					Type: remote.MechanismType_VXLAN,
					Parameters: map[string]string{
						"src_ip": "192.168.111.2",
						"dst_ip": "192.168.111.1",
						"vni":    "127",
					},
				},
			},
		},
	}

	logrus.Infof("Sending request to dataplane: %v", dpAPIConnection)
	rv, err := dataplaneClient.Request(dpCtx, dpAPIConnection)
	if err != nil {
		logrus.Errorf("Dataplane request failed: %s", err)
		return
	}

	srcCon := rv.GetSource().(*crossconnect.CrossConnect_RemoteSource).RemoteSource
	dstCon := rv.GetDestination().(*crossconnect.CrossConnect_RemoteDestination).RemoteDestination
	logrus.Infof("Got Connection: %s <---> %s", srcCon, dstCon)
}
