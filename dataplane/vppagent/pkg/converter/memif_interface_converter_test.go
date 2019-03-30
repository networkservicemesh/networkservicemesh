package converter_test

import (
	"github.com/ligato/vpp-agent/api/models/vpp"
	"github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	. "github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
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

func createTestContext() *connectioncontext.ConnectionContext {
	return &connectioncontext.ConnectionContext{
		SrcIpAddr: srcIp,
		DstIpAddr: dstIp,
	}
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

func checkInterface(intf *vpp.Interface, side ConnectionContextSide) {
	Expect(intf.Name).To(Equal(interfaceName))
	Expect(intf.IpAddresses).To(Equal(ipAddress[side]))
	Expect(intf.Type).To(Equal(vpp_interfaces.Interface_MEMIF))
}

func checkMemif(memif *vpp_interfaces.Interface_Memif, isMaster bool) {
	Expect(memif.Memif.Master).To(Equal(isMaster))
	Expect(memif.Memif.SocketFilename).To(Equal(path.Join(baseDir, mechanismName, socketFilename)))
}

func TestSourceSideConverter(t *testing.T) {
	RegisterTestingT(t)
	conversionParameters := &ConnectionConversionParameters{
		Terminate: true,
		Side:      SOURCE,
		Name:      interfaceName,
		BaseDir:   baseDir,
	}
	converter := NewMemifInterfaceConverter(createTestConnection(), conversionParameters)
	dataRequest, err := converter.ToDataRequest(nil, true)
	Expect(err).To(BeNil())

	Expect(dataRequest.VppConfig.Interfaces).ToNot(BeEmpty())
	checkInterface(dataRequest.VppConfig.Interfaces[0], SOURCE)

	Expect(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif)).ToNot(BeNil())
	checkMemif(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif), false)
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
	dataRequest, err := converter.ToDataRequest(nil, true)
	Expect(err).To(BeNil())

	Expect(dataRequest.VppConfig.Interfaces).ToNot(BeEmpty())
	checkInterface(dataRequest.VppConfig.Interfaces[0], NEITHER)

	Expect(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif)).ToNot(BeNil())
	checkMemif(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif), false)
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
	dataRequest, err := converter.ToDataRequest(nil, true)
	Expect(err).To(BeNil())

	Expect(dataRequest.VppConfig.Interfaces).ToNot(BeEmpty())
	checkInterface(dataRequest.VppConfig.Interfaces[0], DESTINATION)

	Expect(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif).Memif).ToNot(BeNil())
	checkMemif(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif), true)

	os.RemoveAll(baseDir)
}
