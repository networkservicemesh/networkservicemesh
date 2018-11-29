package converter_test

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	. "github.com/ligato/networkservicemesh/dataplane/vppagent/pkg/converter"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	. "github.com/onsi/gomega"
	"os"
	"path"
	"testing"
)

const (
	networkService       = "test-network-service"
	mechanismName        = "test-mechanism"
	mechanismDescription = "test mechanism description"
	socketFilename       = "test.sock"
	connectionId         = "1"
	interfaceName        = "test-interface"
	baseDir              = "./tmp-test-vpp-memif"
	srcIp                = "10.30.1.1/30"
	dstIp                = "10.30.1.2/30"
)

func createTestMechanism() *connection.Mechanism {
	return &connection.Mechanism{
		Type: connection.MechanismType_MEM_INTERFACE,
		Parameters: map[string]string{
			connection.InterfaceNameKey:        mechanismName,
			connection.InterfaceDescriptionKey: mechanismDescription,
			connection.SocketFilename:          path.Join(mechanismName, socketFilename),
		},
	}
}

func createTestContext() map[string]string {
	connectionContext := make(map[string]string)

	connectionContext["src_ip"] = srcIp
	connectionContext["dst_ip"] = dstIp

	return connectionContext
}

func createTestConnection() *connection.Connection {
	return &connection.Connection{
		Id:             connectionId,
		NetworkService: networkService,
		Mechanism:      createTestMechanism(),
		Context:        createTestContext(),
	}
}

var ipAddress = map[ConnectionContextSide][]string{
	SOURCE:      {srcIp},
	DESTINATION: {dstIp},
}

func checkInterface(intf *interfaces.Interfaces_Interface, side ConnectionContextSide) {
	Expect(intf.Name).To(Equal(interfaceName))
	Expect(intf.IpAddresses).To(Equal(ipAddress[side]))
	Expect(intf.Type).To(Equal(interfaces.InterfaceType_MEMORY_INTERFACE))
}

func checkMemif(memif *interfaces.Interfaces_Interface_Memif, isMaster bool) {
	Expect(memif.Master).To(Equal(isMaster))
	Expect(memif.SocketFilename).To(Equal(path.Join(baseDir, mechanismName, socketFilename)))
}

func TestSourceSideConverter(t *testing.T) {
	RegisterTestingT(t)
	conversionParameters := &ConnectionConversionParameters{
		Terminate: false,
		Side:      SOURCE,
		Name:      interfaceName,
		BaseDir:   baseDir,
	}
	converter := NewMemifInterfaceConverter(createTestConnection(), conversionParameters)
	dataRequest, err := converter.ToDataRequest(nil)
	Expect(err).To(BeNil())

	Expect(dataRequest.Interfaces).ToNot(BeEmpty())
	checkInterface(dataRequest.Interfaces[0], SOURCE)

	Expect(dataRequest.Interfaces[0].Memif).ToNot(BeNil())
	checkMemif(dataRequest.Interfaces[0].Memif, false)
}

func TestDestinationSideConverter(t *testing.T) {
	RegisterTestingT(t)
	conversionParameters := &ConnectionConversionParameters{
		Terminate: false,
		Side:      DESTINATION,
		Name:      interfaceName,
		BaseDir:   baseDir,
	}
	converter := NewMemifInterfaceConverter(createTestConnection(), conversionParameters)
	dataRequest, err := converter.ToDataRequest(nil)
	Expect(err).To(BeNil())

	Expect(dataRequest.Interfaces).ToNot(BeEmpty())
	checkInterface(dataRequest.Interfaces[0], NEITHER)

	Expect(dataRequest.Interfaces[0].Memif).ToNot(BeNil())
	checkMemif(dataRequest.Interfaces[0].Memif, false)
}

func TestTerminateDestinationSideConverter(t *testing.T) {
	RegisterTestingT(t)
	conversionParameters := &ConnectionConversionParameters{
		Terminate: true,
		Side:      DESTINATION,
		Name:      interfaceName,
		BaseDir:   baseDir,
	}
	converter := NewMemifInterfaceConverter(createTestConnection(), conversionParameters)
	dataRequest, err := converter.ToDataRequest(nil)
	Expect(err).To(BeNil())

	Expect(dataRequest.Interfaces).ToNot(BeEmpty())
	checkInterface(dataRequest.Interfaces[0], DESTINATION)

	Expect(dataRequest.Interfaces[0].Memif).ToNot(BeNil())
	checkMemif(dataRequest.Interfaces[0].Memif, true)

	os.RemoveAll(baseDir)
}
