package endpoint

import (
	"context"
	"net"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

//NewMacEndpoint creates a new endpoint that adds ethernet context with specific mac addresses
func NewMacEndpoint(srcMac, dstMac string) networkservice.NetworkServiceServer {
	return &mac{srcMac, dstMac}
}

type mac struct {
	dstMac string
	srcMac string
}

func (m *mac) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (m *mac) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	c := request.GetConnection()
	if c.GetContext() != nil {
		c.GetContext().EthernetContext = &connectioncontext.EthernetContext{}
		if m.srcMac != "" {
			c.GetContext().GetEthernetContext().SrcMacAddress = m.srcMac
		}
		if m.dstMac != "" {
			c.GetContext().GetEthernetContext().SrcMacAddress = m.dstMac
		} else {
			ip := ""
			if c.GetContext().GetIpContext() != nil {
				ip = strings.Split(c.Context.IpContext.DstIpAddr, "/")[0]
			}
			c.GetContext().GetEthernetContext().DstMacAddress = generateMacBaseOnIP(ip)
		}
		if err := c.GetContext().GetEthernetContext().Validate(); err != nil {
			return request.GetConnection(), err
		}
	}
	return request.GetConnection(), nil
}

func (m *mac) Name() string {
	return "Mac Address Mutator Endpoint"
}

func generateMacBaseOnIP(ip string) string {
	ipAddr := net.ParseIP(ip).To4()
	mac := administeredAddress([]byte(ipAddr))
	return net.HardwareAddr(mac).String()
}

//http://www.noah.org/wiki/MAC_address#locally_administered_address
func administeredAddress(input []byte) []byte {
	result := make([]byte, 6)
	or := []byte{2, 0, 0, 0, 0, 0}
	and := []byte{254, 255, 255, 255, 255, 255}
	min := len(result)
	if min > len(input) {
		min = len(input)
	}
	for i := 0; i < min; i++ {
		result[i] = or[i] | input[i]&and[i]
	}
	return result
}
