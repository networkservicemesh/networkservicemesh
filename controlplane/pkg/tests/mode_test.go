package tests

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	. "github.com/onsi/gomega"
	"testing"
)

func newModel() model.Model {
	return model.NewModel()
}

func TestModelAddRemove(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()

	mdl.AddDataplane(&model.Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})

	Expect(mdl.GetDataplane("test_name").RegisteredName).To(Equal("test_name"))

	mdl.DeleteDataplane("test_name")

	Expect(mdl.GetDataplane("test_name")).To(BeNil())
}

func TestModelSelectDataplane(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()

	mdl.AddDataplane(&model.Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})
	dp, err := mdl.SelectDataplane(nil)
	Expect(dp.RegisteredName).To(Equal("test_name"))
	Expect(err).To(BeNil())
}
func TestModelSelectDataplaneNone(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()

	dp, err := mdl.SelectDataplane(nil)
	Expect(dp).To(BeNil())
	Expect(err.Error()).To(Equal("no appropriate dataplanes found"))
}

func TestModelAddEndpoint(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()

	ep1 := createNSERegistration("golden-network", "ep1")
	mdl.AddEndpoint(ep1)
	Expect(mdl.GetEndpoint("ep1")).To(Equal(ep1))

	Expect(mdl.GetNetworkServiceEndpoints("golden-network")[0]).To(Equal(ep1))
}

func createNSERegistration(networkServiceName string, endpointName string) *model.Endpoint {
	return &model.Endpoint{
		SocketLocation: "none",
		Workspace:      "nsm-1",
		Endpoint: &registry.NSERegistration{
			NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
				NetworkServiceName: networkServiceName,
				EndpointName:       endpointName,
			}, NetworkService: &registry.NetworkService{
				Name: networkServiceName,
			},
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

func (impl *ListenerImpl) ClientConnectionUpdated(clientConnection *model.ClientConnection) {
}

func (impl *ListenerImpl) ClientConnectionAdded(clientConnection *model.ClientConnection) {
}

func (impl *ListenerImpl) ClientConnectionDeleted(clientConnection *model.ClientConnection) {
}

func (impl *ListenerImpl) EndpointAdded(endpoint *model.Endpoint) {
	impl.endpoints++
}

func (impl *ListenerImpl) EndpointDeleted(endpoint *model.Endpoint) {
	impl.endpoints--
}

func (impl *ListenerImpl) DataplaneAdded(dataplane *model.Dataplane) {
	impl.dataplanes++
}

func (impl *ListenerImpl) DataplaneDeleted(dataplane *model.Dataplane) {
	impl.dataplanes--
}

func TestModelListeners(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()
	listener := &ListenerImpl{}
	mdl.AddListener(listener)

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	mdl.RemoveListener(listener)
}

func TestModelListenDataplane(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()
	listener := &ListenerImpl{}
	mdl.AddListener(listener)

	mdl.AddDataplane(&model.Dataplane{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})

	Expect(listener.dataplanes).To(Equal(1))
	Expect(listener.endpoints).To(Equal(0))

	mdl.DeleteDataplane("test_name")

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	mdl.RemoveListener(listener)
}

func TestModelListenEndpoint(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()
	listener := &ListenerImpl{}
	mdl.AddListener(listener)

	mdl.AddEndpoint(createNSERegistration("golden-network", "ep1"))
	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(1))

	_ = mdl.DeleteEndpoint("ep1")

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	mdl.RemoveListener(listener)
}

func TestModelListenExistingEndpoint(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()
	listener := &ListenerImpl{}

	mdl.AddEndpoint(createNSERegistration("golden-network", "ep1"))

	// Since model will call for all existing, this should be same
	mdl.AddListener(listener)

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(1))

	_ = mdl.DeleteEndpoint("ep1")

	Expect(listener.dataplanes).To(Equal(0))
	Expect(listener.endpoints).To(Equal(0))

	mdl.RemoveListener(listener)
}

func TestModelRestoreIds(t *testing.T) {
	RegisterTestingT(t)

	mdl := newModel()
	Expect(mdl.ConnectionId()).To(Equal("1"))
	Expect(mdl.ConnectionId()).To(Equal("2"))
	mdl2 := newModel()
	mdl2.CorrectIdGenerator(mdl.ConnectionId())
	Expect(mdl2.ConnectionId()).To(Equal("4"))

}
