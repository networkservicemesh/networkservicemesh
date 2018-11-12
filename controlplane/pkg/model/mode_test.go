package model

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	. "github.com/onsi/gomega"
	"testing"
)

func TestModelAddRemove(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")

	model.AddDataplane(&Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})

	Expect(model.GetDataplane("test_name").RegisteredName).To(Equal("test_name"))

	model.DeleteDataplane("test_name")

	Expect(model.GetDataplane("test_name")).To(BeNil())
}

func TestModelSelectDataplane(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")

	model.AddDataplane(&Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})
	dp, err := model.SelectDataplane()
	Expect(dp.RegisteredName).To(Equal("test_name"))
	Expect(err).To(BeNil())
}
func TestModelSelectDataplaneNone(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")

	dp, err := model.SelectDataplane()
	Expect(dp).To(BeNil())
	Expect(err.Error()).To(Equal("no dataplanes registered"))
}

func TestModelAddEndpoint(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")

	ep1 := &registry.NetworkServiceEndpoint{
		NetworkServiceName: "golden-network",
		EndpointName:       "ep1",
	}
	model.AddEndpoint(ep1)
	Expect(model.GetEndpoint("ep1")).To(Equal(ep1))

	Expect(model.GetNetworkServiceEndpoints("golden-network")[0]).To(Equal(ep1))
}

func TestModelTwoEndpoint(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")

	ep1 := &registry.NetworkServiceEndpoint{
		NetworkServiceName: "golden-network",
		EndpointName:       "ep1",
	}
	ep2 := &registry.NetworkServiceEndpoint{
		NetworkServiceName: "golden-network",
		EndpointName:       "ep2",
	}
	model.AddEndpoint(ep1)
	model.AddEndpoint(ep2)
	Expect(model.GetEndpoint("ep1")).To(Equal(ep1))
	Expect(model.GetEndpoint("ep2")).To(Equal(ep2))

	Expect(len(model.GetNetworkServiceEndpoints("golden-network"))).To(Equal(2))
}

func TestModelAddDeleteEndpoint(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")

	ep1 := &registry.NetworkServiceEndpoint{
		NetworkServiceName: "golden-network",
		EndpointName:       "ep1",
	}
	ep2 := &registry.NetworkServiceEndpoint{
		NetworkServiceName: "golden-network",
		EndpointName:       "ep2",
	}
	model.AddEndpoint(ep1)
	model.AddEndpoint(ep2)
	model.DeleteEndpoint("ep1")
	Expect(model.GetEndpoint("ep1")).To(BeNil())
	Expect(model.GetEndpoint("ep2")).To(Equal(ep2))

	Expect(len(model.GetNetworkServiceEndpoints("golden-network"))).To(Equal(1))
}
