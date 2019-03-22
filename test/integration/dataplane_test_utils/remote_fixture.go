package dataplane_test_utils

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"strconv"
	"time"
)

// A standaloneDataplaneRemoteFixture represents minimalist test configuration
// with just two data-planes and two peers (source and destination)
// deployed on two nodes respectively.
type StandaloneDataplaneRemoteFixture struct {
	timeout         time.Duration
	k8s             *kube_testing.K8s
	sourcePod       *v1.Pod
	destPod         *v1.Pod
	sourceDataplane *StandaloneDataplaneInstance
	destDataplane   *StandaloneDataplaneInstance
	vni             int
}

func CreateRemoteFixture(timeout time.Duration) *StandaloneDataplaneRemoteFixture {
	fixture := &StandaloneDataplaneRemoteFixture{
		timeout: timeout,
		vni:     1,
	}

	k8s, err := kube_testing.NewK8s()
	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "dataplane", "icmp-responder-nse", "jaeger", "source-pod", "destination-pod")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	// prepare node
	nodes := k8s.GetNodesWait(2, timeout)
	Expect(len(nodes) >= 2).To(Equal(true), "At least one kubernetes node is required for this test")

	fixture.k8s = k8s
	sourceNode := &nodes[0]
	destNode := &nodes[1]

	fixture.sourceDataplane = CreateDataplaneInstance(k8s, sourceNode, timeout)
	fixture.destDataplane = CreateDataplaneInstance(k8s, destNode, timeout)

	// deploy source and destination pods
	fixture.sourcePod = k8s.CreatePod(pods.AlpinePod("source-pod", sourceNode))
	fixture.destPod = k8s.CreatePod(pods.AlpinePod("destination-pod", destNode))

	return fixture
}

func (fixture *StandaloneDataplaneRemoteFixture) Cleanup() {
	fixture.sourceDataplane.Cleanup()
	fixture.destDataplane.Cleanup()
	fixture.k8s.Cleanup()
}

func (fixture *StandaloneDataplaneRemoteFixture) SourceDataplane() *StandaloneDataplaneInstance {
	return fixture.sourceDataplane
}

func (fixture *StandaloneDataplaneRemoteFixture) DestDataplane() *StandaloneDataplaneInstance {
	return fixture.destDataplane
}

func (fixture *StandaloneDataplaneRemoteFixture) RequestKernelConnectionArp(id, iface, srcIp, dstIp string, neighbors []*connectioncontext.IpNeighbor) (*crossconnect.CrossConnect, *crossconnect.CrossConnect) {
	sourceNetNs := GetNetNS(fixture.k8s, fixture.sourcePod)
	destNetNs := GetNetNS(fixture.k8s, fixture.destPod)
	vxlan := fixture.createVxlan(fixture.nextVni())

	connSrc := CreateRemoteXConnectRequestSrc(id, "kernel", vxlan, iface, srcIp, dstIp, sourceNetNs, neighbors)
	connDst := CreateRemoteXConnectRequestDst(id, vxlan, "kernel", iface, srcIp, dstIp, destNetNs, neighbors)

	connSrc = fixture.sourceDataplane.Request(connSrc)
	connDst = fixture.destDataplane.Request(connDst)
	return connSrc, connDst
}

func (fixture *StandaloneDataplaneRemoteFixture) RequestKernelConnection(id, iface, srcIp, dstIp string) (*crossconnect.CrossConnect, *crossconnect.CrossConnect) {
	return fixture.RequestKernelConnectionArp(id, iface, srcIp, dstIp, []*connectioncontext.IpNeighbor{})
}

func (fixture *StandaloneDataplaneRemoteFixture) RequestDefaultKernelConnection() (*crossconnect.CrossConnect, *crossconnect.CrossConnect) {
	return fixture.RequestKernelConnection("some-id", "iface", "10.30.1.1/30", "10.30.1.2/30")
}

func (fixture *StandaloneDataplaneRemoteFixture) VerifyKernelConnectionSrc(conn *crossconnect.CrossConnect) {
	srcIface := getIface(conn.GetLocalSource())
	srcIp := unmaskIp(conn.GetLocalSource().Context.SrcIpAddr)
	dstIp := unmaskIp(conn.GetLocalSource().Context.DstIpAddr)
	VerifyKernelConnectionEstablished(fixture.k8s, fixture.sourcePod, srcIface, srcIp, dstIp)
}

func (fixture *StandaloneDataplaneRemoteFixture) VerifyKernelConnectionDst(conn *crossconnect.CrossConnect) {
	dstIface := getIface(conn.GetLocalDestination())
	srcIp := unmaskIp(conn.GetLocalDestination().Context.SrcIpAddr)
	dstIp := unmaskIp(conn.GetLocalDestination().Context.DstIpAddr)
	VerifyKernelConnectionEstablished(fixture.k8s, fixture.destPod, dstIface, dstIp, srcIp)
}

func (fixture *StandaloneDataplaneRemoteFixture) VerifyKernelConnection(connSrc, connDst *crossconnect.CrossConnect) {
	fixture.VerifyKernelConnectionSrc(connSrc)
	fixture.VerifyKernelConnectionDst(connDst)
}

func (fixture *StandaloneDataplaneRemoteFixture) VerifyKernelConnectionClosedSrc(conn *crossconnect.CrossConnect) {
	srcIface := getIface(conn.GetLocalSource())
	VerifyKernelConnectionClosed(fixture.k8s, fixture.sourcePod, srcIface)
}

func (fixture *StandaloneDataplaneRemoteFixture) VerifyKernelConnectionClosedDst(conn *crossconnect.CrossConnect) {
	dstIface := getIface(conn.GetLocalDestination())
	VerifyKernelConnectionClosed(fixture.k8s, fixture.destPod, dstIface)
}

func (fixture *StandaloneDataplaneRemoteFixture) VerifyKernelConnectionClosed(connSrc, connDst *crossconnect.CrossConnect) {
	fixture.VerifyKernelConnectionClosedSrc(connSrc)
	fixture.VerifyKernelConnectionClosedDst(connDst)
}

func (fixture *StandaloneDataplaneRemoteFixture) createVxlan(vni string) *remote.Mechanism {
	srcIp := fixture.SourceDataplane().Pod().Status.PodIP
	dstIp := fixture.DestDataplane().Pod().Status.PodIP
	return vxlanMechanism(srcIp, dstIp, vni)
}

func (fixture *StandaloneDataplaneRemoteFixture) nextVni() string {
	vni := strconv.FormatInt(int64(fixture.vni), 10)
	fixture.vni += 1
	return vni
}

func (fixture *StandaloneDataplaneRemoteFixture) updateVxlan(conn *crossconnect.CrossConnect) {
	if src, ok := conn.GetSource().(*crossconnect.CrossConnect_RemoteSource); ok {
		vni := vxlanVni(src.RemoteSource.Mechanism)
		src.RemoteSource.Mechanism = fixture.createVxlan(vni)
	}

	if dst, ok := conn.GetDestination().(*crossconnect.CrossConnect_RemoteDestination); ok {
		vni := vxlanVni(dst.RemoteDestination.Mechanism)
		dst.RemoteDestination.Mechanism = fixture.createVxlan(vni)
	}
}

func (fixture *StandaloneDataplaneRemoteFixture) HealConnectionSrc(connSrc *crossconnect.CrossConnect) *crossconnect.CrossConnect {
	fixture.updateVxlan(connSrc)
	return fixture.SourceDataplane().Request(connSrc)
}

func (fixture *StandaloneDataplaneRemoteFixture) HealConnectionDst(connDst *crossconnect.CrossConnect) *crossconnect.CrossConnect {
	fixture.updateVxlan(connDst)
	return fixture.DestDataplane().Request(connDst)
}

func (fixture *StandaloneDataplaneRemoteFixture) HealConnection(connSrc, connDst *crossconnect.CrossConnect) (*crossconnect.CrossConnect, *crossconnect.CrossConnect) {
	return fixture.HealConnectionSrc(connSrc), fixture.HealConnectionDst(connDst)
}
