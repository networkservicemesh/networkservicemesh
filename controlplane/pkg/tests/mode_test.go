// +build unit_test

package tests

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

func newModel() model.Model {
	return model.NewModel()
}

func TestModelAddRemove(t *testing.T) {
	g := NewWithT(t)

	mdl := newModel()

	mdl.AddForwarder(context.Background(), &model.Forwarder{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})

	g.Expect(mdl.GetForwarder("test_name").RegisteredName).To(Equal("test_name"))

	mdl.DeleteForwarder(context.Background(), "test_name")

	g.Expect(mdl.GetForwarder("test_name")).To(BeNil())
}

func TestModelSelectForwarder(t *testing.T) {
	g := NewWithT(t)

	mdl := newModel()

	mdl.AddForwarder(context.Background(), &model.Forwarder{
		RegisteredName: "test_name",
		SocketLocation: "location",
	})
	dp, err := mdl.SelectForwarder(nil)
	g.Expect(dp.RegisteredName).To(Equal("test_name"))
	g.Expect(err).To(BeNil())
}
func TestModelSelectForwarderNone(t *testing.T) {
	g := NewWithT(t)

	mdl := newModel()

	dp, err := mdl.SelectForwarder(nil)
	g.Expect(dp).To(BeNil())
	g.Expect(err.Error()).To(Equal("no appropriate forwarders found"))
}

func TestModelAddEndpoint(t *testing.T) {
	g := NewWithT(t)

	mdl := newModel()

	ep1 := createNSERegistration("golden-network", "ep1")
	mdl.AddEndpoint(context.Background(), ep1)
	g.Expect(mdl.GetEndpoint("ep1")).To(Equal(ep1))

	g.Expect(mdl.GetEndpointsByNetworkService("golden-network")[0]).To(Equal(ep1))
}

func createNSERegistration(networkServiceName string, endpointName string) *model.Endpoint {
	return &model.Endpoint{
		SocketLocation: "none",
		Workspace:      "nsm-1",
		Endpoint: &registry.NSERegistration{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name:               endpointName,
				NetworkServiceName: networkServiceName,
			}, NetworkService: &registry.NetworkService{
				Name: networkServiceName,
			},
		},
	}
}

func TestModelTwoEndpoint(t *testing.T) {
	g := NewWithT(t)

	model := newModel()

	ep1 := createNSERegistration("golden-network", "ep1")
	ep2 := createNSERegistration("golden-network", "ep2")
	model.AddEndpoint(context.Background(), ep1)
	model.AddEndpoint(context.Background(), ep2)
	g.Expect(model.GetEndpoint("ep1")).To(Equal(ep1))
	g.Expect(model.GetEndpoint("ep2")).To(Equal(ep2))

	g.Expect(len(model.GetEndpointsByNetworkService("golden-network"))).To(Equal(2))
}

func TestModelAddDeleteEndpoint(t *testing.T) {
	g := NewWithT(t)

	model := newModel()

	ep1 := createNSERegistration("golden-network", "ep1")
	ep2 := createNSERegistration("golden-network", "ep2")
	model.AddEndpoint(context.Background(), ep1)
	model.AddEndpoint(context.Background(), ep2)
	model.DeleteEndpoint(context.Background(), "ep1")
	g.Expect(model.GetEndpoint("ep1")).To(BeNil())
	g.Expect(model.GetEndpoint("ep2")).To(Equal(ep2))

	g.Expect(len(model.GetEndpointsByNetworkService("golden-network"))).To(Equal(1))
}

func TestModelRestoreIds(t *testing.T) {
	g := NewWithT(t)

	mdl := newModel()
	g.Expect(mdl.ConnectionID()).To(Equal("1"))
	g.Expect(mdl.ConnectionID()).To(Equal("2"))
	mdl2 := newModel()
	mdl2.CorrectIDGenerator(mdl.ConnectionID())
	g.Expect(mdl2.ConnectionID()).To(Equal("4"))
}
