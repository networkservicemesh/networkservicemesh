package tests

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	mdl "github.com/ligato/networkservicemesh/controlplane/pkg/model"
	. "github.com/onsi/gomega"
	"testing"
)

func newModel() mdl.Model {
	return mdl.NewModel()
}

func TestModelAddRemove(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()

	model.AddDataplane(&mdl.Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})

	Expect(model.GetDataplane("test_name").RegisteredName).To(Equal("test_name"))

	model.DeleteDataplane("test_name")

	Expect(model.GetDataplane("test_name")).To(BeNil())
}

func TestModelSelectDataplane(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()

	model.AddDataplane(&mdl.Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})
	dp, err := model.SelectDataplane()
	Expect(dp.RegisteredName).To(Equal("test_name"))
	Expect(err).To(BeNil())
}
func TestModelSelectDataplaneNone(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()

	dp, err := model.SelectDataplane()
	Expect(dp).To(BeNil())
	Expect(err.Error()).To(Equal("no dataplanes registered"))
}

func TestModelAddEndpoint(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()

	ep1 := createNSERegistration("golden-network", "ep1")
	model.AddEndpoint(ep1)
	Expect(model.GetEndpoint("ep1")).To(Equal(ep1))

	Expect(model.GetNetworkServiceEndpoints("golden-network")[0]).To(Equal(ep1))
}

func createNSERegistration(networkServiceName string, endpointName string) *registry.NSERegistration {
	return &registry.NSERegistration{
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: networkServiceName,
			EndpointName:       endpointName,
		}, NetworkService: &registry.NetworkService{
			Name: networkServiceName,
		},
	}
}

func TestModelTwoEndpoint(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()

	ep1 := createNSERegistration("golden-network", "ep1")
	ep2 := createNSERegistration("golden-network", "ep2")
	model.AddEndpoint(ep1)
	model.AddEndpoint(ep2)
	Expect(model.GetEndpoint("ep1")).To(Equal(ep1))
	Expect(model.GetEndpoint("ep2")).To(Equal(ep2))

	Expect(len(model.GetNetworkServiceEndpoints("golden-network"))).To(Equal(2))
}

func TestModelAddDeleteEndpoint(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()

	ep1 := createNSERegistration("golden-network", "ep1")
	ep2 := createNSERegistration("golden-network", "ep2")
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

func (impl *ListenerImpl) DataplaneAdded(dataplane *mdl.Dataplane) {
	impl.dataplanes++
}

func (impl *ListenerImpl) DataplaneDeleted(dataplane *mdl.Dataplane) {
	impl.dataplanes--
}

func TestModelListeners(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()
	listener := &ListenerImpl{}
	model.AddListener(listener)

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	model.RemoveListener(listener)
}

func TestModelListenDataplane(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()
	listener := &ListenerImpl{}
	model.AddListener(listener)

	model.AddDataplane(&mdl.Dataplane{
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

	model := newModel()
	listener := &ListenerImpl{}
	model.AddListener(listener)

	model.AddEndpoint(&registry.NSERegistration{
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: "golden-network",
			EndpointName:       "ep1",
		},
	})

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(1))

	model.DeleteEndpoint("ep1")

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	model.RemoveListener(listener)
}

func TestModelListenExistingEndpoint(t *testing.T) {
	RegisterTestingT(t)

	model := newModel()
	listener := &ListenerImpl{}

	model.AddEndpoint(&registry.NSERegistration{
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: "golden-network",
			EndpointName:       "ep1",
		},
	})

	// Since model will call for all existing, this should be same
	model.AddListener(listener)

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(1))

	model.DeleteEndpoint("ep1")

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	model.RemoveListener(listener)
}
