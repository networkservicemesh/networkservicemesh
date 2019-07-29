package tests

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nseregistry"
)

func TestNSEFileRegistry(t *testing.T) {
	g := NewWithT(t)
	fileName, err := tmpFile()
	g.Expect(err).To(BeNil())
	defer os.Remove(fileName)
	reg := nseregistry.NewNSERegistry(fileName)

	err = addValues(reg)
	g.Expect(err).To(BeNil())

	clients, nses, err := reg.LoadRegistry()
	g.Expect(err).To(BeNil())
	g.Expect(clients).To(Equal([]string{"nsm-1", "nsm-2", "nsm-3"}))
	g.Expect(nses).To(Equal(map[string]nseregistry.NSEEntry{"endpoint1": createEntry("nsm-1", "endpoint1"), "endpoint2": createEntry("nsm-2", "endpoint2")}))
}
func createEntry(workspace string, endpoint string) nseregistry.NSEEntry {
	return nseregistry.NSEEntry{
		Workspace: workspace,
		NseReg:    createNSEReg(endpoint),
	}
}
func TestNSEDeleteTest(t *testing.T) {
	g := NewWithT(t)
	fileName, err := tmpFile()
	g.Expect(err).To(BeNil())
	defer os.Remove(fileName)
	reg := nseregistry.NewNSERegistry(fileName)

	err = addValues(reg)
	g.Expect(err).To(BeNil())

	g.Expect(reg.DeleteNSE("endpoint1")).To(BeNil())

	clients, nses, err := reg.LoadRegistry()
	g.Expect(err).To(BeNil())
	g.Expect(clients).To(Equal([]string{"nsm-1", "nsm-2", "nsm-3"}))
	g.Expect(nses).To(Equal(map[string]nseregistry.NSEEntry{"endpoint2": createEntry("nsm-2", "endpoint2")}))
}

func TestNSEDeleteClientTest(t *testing.T) {
	g := NewWithT(t)
	fileName, err := tmpFile()
	g.Expect(err).To(BeNil())
	defer os.Remove(fileName)
	reg := nseregistry.NewNSERegistry(fileName)

	err = addValues(reg)
	g.Expect(err).To(BeNil())

	g.Expect(reg.DeleteClient("nsm-1")).To(BeNil())

	clients, nses, err := reg.LoadRegistry()
	g.Expect(err).To(BeNil())
	g.Expect(clients).To(Equal([]string{"nsm-2", "nsm-3"}))
	g.Expect(nses).To(Equal(map[string]nseregistry.NSEEntry{"endpoint2": nseregistry.NSEEntry{
		Workspace: "nsm-2",
		NseReg:    createNSEReg("endpoint2"),
	}}))
}

func addValues(reg *nseregistry.NSERegistry) error {
	err := reg.AppendClientRequest("nsm-1")
	if err != nil {
		return err
	}
	err = reg.AppendClientRequest("nsm-2")
	if err != nil {
		return err
	}
	err = reg.AppendClientRequest("nsm-3")
	if err != nil {
		return err
	}
	err = reg.AppendNSERegRequest("nsm-1", createNSEReg("endpoint1"))
	if err != nil {
		return err
	}
	err = reg.AppendNSERegRequest("nsm-2", createNSEReg("endpoint2"))
	return err
}

func createNSEReg(name string) *registry.NSERegistration {
	return &registry.NSERegistration{
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			EndpointName:              name,
			NetworkServiceManagerName: "nsm1",
		},
		NetworkService: &registry.NetworkService{
			Name:    "my_nsm",
			Payload: "A\nB",
		},
	}
}

func tmpFile() (string, error) {
	regFile, err := ioutil.TempFile(os.TempDir(), "nsm_reg.data")
	if err != nil {
		return "", err
	}
	fileName := regFile.Name()
	_ = regFile.Close()
	_ = os.Remove(fileName)
	return fileName, nil
}
