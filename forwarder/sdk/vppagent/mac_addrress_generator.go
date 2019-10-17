package forwarder

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

//NewDestinationMacAddressGenerator simulates generation phys addr on vpp side...
func NewDestinationMacAddressGenerator() forwarder.ForwarderServer {
	return &destinationMacAddressGenerator{}
}

type destinationMacAddressGenerator struct {
}

func (c *destinationMacAddressGenerator) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	mac := c.generateMac()
	if crossConnect.GetLocalDestination() != nil {
		crossConnect.GetLocalDestination().GetContext().EthernetContext = &connectioncontext.EthernetContext{
			DstMac: mac,
		}
	}

	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(ctx, crossConnect)
}

func (c *destinationMacAddressGenerator) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

func (c *destinationMacAddressGenerator) generateMac() string {
	bytes := make([]byte, 6)
	_, _ = rand.Read(bytes)
	bytes = administeredAddress(bytes)
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5])
}

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
