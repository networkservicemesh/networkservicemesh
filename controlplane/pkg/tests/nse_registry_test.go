package tests

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nseregistry"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"testing"
)

func TestNSEFileRegistry(t *testing.T) {
	RegisterTestingT(t)
	fileName := tmpFile()
	defer os.Remove(fileName)
	reg := nseregistry.NewNSERegistry(fileName)

	addValues(reg)

	clients, nses, err := reg.LoadRegistry()
	Expect(err).To(BeNil())
	Expect(clients).To(Equal([]string{"nsm-1", "nsm-2", "nsm-3"}))
	Expect(nses).To(Equal(map[string]nseregistry.NSEEntry{"endpoint1": createEntry("nsm-1", "endpoint1"), "endpoint2": createEntry("nsm-2", "endpoint2")}))
}
func createEntry(workspace string, endpoint string) nseregistry.NSEEntry {
	return nseregistry.NSEEntry {
		Workspace: workspace,
		NseReg: createNSEReg(endpoint),
	}
}
func TestNSEDeleteTest(t *testing.T) {
	RegisterTestingT(t)
	fileName := tmpFile()
	defer os.Remove(fileName)
	reg := nseregistry.NewNSERegistry(fileName)

	addValues(reg)

	Expect(reg.DeleteNSE("endpoint1")).To(BeNil())

	clients, nses, err := reg.LoadRegistry()
	Expect(err).To(BeNil())
	Expect(clients).To(Equal([]string{"nsm-1", "nsm-2", "nsm-3"}))
	Expect(nses).To(Equal(map[string]nseregistry.NSEEntry {"endpoint2": createEntry("nsm-2", "endpoint2")}))
}

func TestNSEDeleteClientTest(t *testing.T) {
	RegisterTestingT(t)
	fileName := tmpFile()
	defer os.Remove(fileName)
	reg := nseregistry.NewNSERegistry(fileName)


	addValues(reg)

	Expect(reg.DeleteClient("nsm-1")).To(BeNil())

	clients, nses, err := reg.LoadRegistry()
	Expect(err).To(BeNil())
	Expect(clients).To(Equal([]string{"nsm-2", "nsm-3"}))
	Expect(nses).To(Equal(map[string]nseregistry.NSEEntry{"endpoint2": nseregistry.NSEEntry {
		Workspace: "nsm-2",
		NseReg: createNSEReg("endpoint2"),
	}}))
}



func addValues(reg *nseregistry.NSERegistry) {
	err := reg.AppendClientRequest("nsm-1")
	Expect(err).To(BeNil())
	err = reg.AppendClientRequest("nsm-2")
	Expect(err).To(BeNil())
	err = reg.AppendClientRequest("nsm-3")
	Expect(err).To(BeNil())
	err = reg.AppendNSERegRequest("nsm-1", createNSEReg("endpoint1"))
	Expect(err).To(BeNil())
	err = reg.AppendNSERegRequest("nsm-2", createNSEReg("endpoint2"))
	Expect(err).To(BeNil())
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

func tmpFile() (string) {
	regFile, err := ioutil.TempFile(os.TempDir(), "nsm_reg.data")
	fileName := regFile.Name()
	Expect(err).To(BeNil())
	_ = regFile.Close()
	_ = os.Remove(fileName)
	return fileName
}
