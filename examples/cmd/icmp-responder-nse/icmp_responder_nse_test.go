package main

import (
	"context"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestRequest(t *testing.T) {
	RegisterTestingT(t)

	service := New("10.20.1.0/24").(*networkService)
	service.netNS = "2"

	response, error := service.Request(context.Background(), createRequest("c1", "my_service"))
	Expect(error).To(BeNil())
	Expect(response.Context.SrcIpAddr).To(Equal("10.20.1.1/30"))
	Expect(response.Context.DstIpAddr).To(Equal("10.20.1.2/30"))
	Expect(response.Context.DstIpRequired).To(Equal(true))
	Expect(response.Context.SrcIpRequired).To(Equal(true))

	Expect(len(response.Context.Routes)).To(Equal(1))
	Expect(response.Context.Routes[0].Prefix).To(Equal("8.8.8.8/30"))

	Expect(len(response.Context.ExtraPrefixes)).To(Equal(1))
	Expect(response.Context.ExtraPrefixes[0]).To(Equal("10.20.1.8/29"))

	// Now lets check if Close() will return all to normal.

	prefixes := service.prefixPool.GetPrefixes();
	Expect(prefixes).To(Equal([]string{"10.20.1.4/30",
		"10.20.1.16/28",
		"10.20.1.32/27",
		"10.20.1.64/26",
		"10.20.1.128/25",}))

	_, error = service.Close(context.Background(), response)
	Expect(error).To(BeNil())
	prefixes = service.prefixPool.GetPrefixes();
	Expect(prefixes).To(Equal([]string{"10.20.1.0/24"}));
	logrus.Printf("End of test %v", response.Id)

}

func createRequest(connectionId string, serviceName string) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					connection.NetNsInodeKey:    "1",
					connection.InterfaceNameKey: "nsm_" + connectionId,
				},
			},
		},
		Connection: &connection.Connection{
			Id:             connectionId,
			NetworkService: serviceName,
			Labels:         map[string]string{},
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
				ExtraPrefixRequest: []*connectioncontext.ExtraPrefixRequest{
					{
						AddrFamily: &connectioncontext.IpFamily{
							Family: connectioncontext.IpFamily_IPV4,
						},
						PrefixLen:       29,
						RequestedNumber: 1,
						RequiredNumber:  1,
					},
				},
			},
		},
	}
}
