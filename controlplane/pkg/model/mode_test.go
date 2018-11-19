package model

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
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

type ListenerImpl struct {
	endpoints  int
	dataplanes int
}

func (impl *ListenerImpl) EndpointAdded(endpoint *registry.NetworkServiceEndpoint) {
	impl.endpoints++
}

func (impl *ListenerImpl) EndpointDeleted(endpoint *registry.NetworkServiceEndpoint) {
	impl.endpoints--
}

func (impl *ListenerImpl) DataplaneAdded(dataplane *Dataplane) {
	impl.dataplanes++
}

func (impl *ListenerImpl) DataplaneDeleted(dataplane *Dataplane) {
	impl.dataplanes--
}

func TestModelListeners(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")
	listener := &ListenerImpl{}
	model.AddListener(listener)

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	model.RemoveListener(listener)
}

func TestModelListenDataplane(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")
	listener := &ListenerImpl{}
	model.AddListener(listener)

	model.AddDataplane(&Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})

	Expect(listener.dataplanes).To(Equal(1))
	Expect(listener.endpoints).To(Equal(0))

	model.DeleteDataplane("test_name")

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	model.RemoveListener(listener)
}

func TestModelListenEndpoint(t *testing.T) {
	RegisterTestingT(t)

	model := NewModel("127.0.0.1:5000")
	listener := &ListenerImpl{}
	model.AddListener(listener)

	model.AddEndpoint(&registry.NetworkServiceEndpoint{
		NetworkServiceName: "golden-network",
		EndpointName:       "ep1",
	})

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(1))

	model.DeleteEndpoint("ep1")

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	model.RemoveListener(listener)
}
