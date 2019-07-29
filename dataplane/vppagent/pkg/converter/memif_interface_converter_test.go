package converter_test

import (
	"os"
	"path"
	"testing"

	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	. "github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
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
		IpContext: &connectioncontext.IPContext{
			SrcIpAddr: srcIp,
			DstIpAddr: dstIp,
		},
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

func checkInterface(g *WithT, intf *vpp.Interface, side ConnectionContextSide) {
	g.Expect(intf.Name).To(Equal(interfaceName))
	g.Expect(intf.IpAddresses).To(Equal(ipAddress[side]))
	g.Expect(intf.Type).To(Equal(vpp_interfaces.Interface_MEMIF))
}

func checkMemif(g *WithT, memif *vpp_interfaces.Interface_Memif, isMaster bool) {
	g.Expect(memif.Memif.Master).To(Equal(isMaster))
	g.Expect(memif.Memif.SocketFilename).To(Equal(path.Join(baseDir, mechanismName, socketFilename)))
}

func TestSourceSideConverter(t *testing.T) {
	g := NewWithT(t)
	conversionParameters := &ConnectionConversionParameters{
		Terminate: true,
		Side:      SOURCE,
		Name:      interfaceName,
		BaseDir:   baseDir,
	}
	converter := NewMemifInterfaceConverter(createTestConnection(), conversionParameters)
	dataRequest, err := converter.ToDataRequest(nil, true)
	g.Expect(err).To(BeNil())

	g.Expect(dataRequest.VppConfig.Interfaces).ToNot(BeEmpty())
	checkInterface(g, dataRequest.VppConfig.Interfaces[0], SOURCE)

	g.Expect(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif)).ToNot(BeNil())
	checkMemif(g, dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif), false)
}

func TestDestinationSideConverter(t *testing.T) {
	g := NewWithT(t)
	conversionParameters := &ConnectionConversionParameters{
		Terminate: false,
		Side:      DESTINATION,
		Name:      interfaceName,
		BaseDir:   baseDir,
	}
	converter := NewMemifInterfaceConverter(createTestConnection(), conversionParameters)
	dataRequest, err := converter.ToDataRequest(nil, true)
	g.Expect(err).To(BeNil())

	g.Expect(dataRequest.VppConfig.Interfaces).ToNot(BeEmpty())
	checkInterface(g, dataRequest.VppConfig.Interfaces[0], NEITHER)

	g.Expect(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif)).ToNot(BeNil())
	checkMemif(g, dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif), false)
}

func TestTerminateDestinationSideConverter(t *testing.T) {
	g := NewWithT(t)
	conversionParameters := &ConnectionConversionParameters{
		Terminate: true,
		Side:      DESTINATION,
		Name:      interfaceName,
		BaseDir:   baseDir,
	}
	converter := NewMemifInterfaceConverter(createTestConnection(), conversionParameters)
	dataRequest, err := converter.ToDataRequest(nil, true)
	g.Expect(err).To(BeNil())

	g.Expect(dataRequest.VppConfig.Interfaces).ToNot(BeEmpty())
	checkInterface(g, dataRequest.VppConfig.Interfaces[0], DESTINATION)

	g.Expect(dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif).Memif).ToNot(BeNil())
	checkMemif(g, dataRequest.VppConfig.Interfaces[0].Link.(*vpp_interfaces.Interface_Memif), true)

	os.RemoveAll(baseDir)
}
