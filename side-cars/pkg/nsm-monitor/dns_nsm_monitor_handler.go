package nsmmonitor

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/utils"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/k8s/api/nsm-coredns/update"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
)

const (
	//UpdateAPIClientSock means path to client socket for dns context update server
	UpdateAPIClientSock = utils.EnvVar("UPDATE_API_CLIENT_SOCKET")
)

//nsmDNSMonitorHandler implements Handler interface for handling dnsConfigs
type nsmDNSMonitorHandler struct {
	EmptyNSMMonitorHandler
	dnsConfigUpdateClient update.DNSConfigServiceClient
}

//NewNsmDNSMonitorHandler creates new DNS monitor handler
func NewNsmDNSMonitorHandler() Handler {
	clientSock := UpdateAPIClientSock.StringValue()
	if clientSock == "" {
		logrus.Fatalf("unable to create Handler instance. Expect %v is not empty", UpdateAPIClientSock.Name())
	}
	conn, err := tools.DialUnix(clientSock)
	if err != nil {
		logrus.Fatalf("An error during dial unix socket by path %v, error: %v", clientSock, err.Error())
	}
	return &nsmDNSMonitorHandler{
		dnsConfigUpdateClient: update.NewDNSConfigServiceClient(conn),
	}
}

func (h *nsmDNSMonitorHandler) Connected(conns map[string]*connection.Connection) {
	for _, conn := range conns {
		if conn.Context == nil || conn.Context.DnsContext == nil {
			continue
		}
		logrus.Info(conn.Context.DnsContext)
		_, _ = h.dnsConfigUpdateClient.AddDNSContext(context.Background(), &update.AddDNSContextMessage{ConnectionID: conn.Id, Context: conn.Context.DnsContext})
	}
}

func (h *nsmDNSMonitorHandler) Closed(conn *connection.Connection) {
	logrus.Infof("Deleting config with id %v", conn.Id)
	_, _ = h.dnsConfigUpdateClient.RemoveDNSContext(context.Background(), &update.RemoveDNSContextMessage{ConnectionID: conn.Id})
}
