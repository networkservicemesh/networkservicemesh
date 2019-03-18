package dataplane_test_utils

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"time"
)

// A standaloneDataplaneFixture represents minimalist test configuration
// with just a dataplane pod and two peer pods (source and destination)
// deployed on a single node.
type StandaloneDataplaneLocalFixture struct {
	timeout   time.Duration
	k8s       *kube_testing.K8s
	Dataplane *StandaloneDataplaneInstance
	sourcePod *v1.Pod
	destPod   *v1.Pod
}

func CreateLocalFixture(timeout time.Duration) *StandaloneDataplaneLocalFixture {
	fixture := &StandaloneDataplaneLocalFixture{
		timeout: timeout,
	}

	k8s, err := kube_testing.NewK8s()
	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "dataplane", "icmp-responder-nse", "jaeger", "source-pod", "destination-pod")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	// prepare node
	nodes := k8s.GetNodesWait(1, timeout)
	Expect(len(nodes) >= 1).To(Equal(true), "At least one kubernetes node is required for this test")

	fixture.k8s = k8s
	node := &nodes[0]

	fixture.Dataplane = CreateDataplaneInstance(k8s, node, timeout)

	// deploy source and destination pods
	fixture.sourcePod = k8s.CreatePod(pods.AlpinePod("source-pod", node))
	fixture.destPod = k8s.CreatePod(pods.AlpinePod("destination-pod", node))

	return fixture
}

func (fixture *StandaloneDataplaneLocalFixture) Cleanup() {
	fixture.Dataplane.Cleanup()
	fixture.k8s.Cleanup()
}

func (fixture *StandaloneDataplaneLocalFixture) RequestKernelConnectionArp(id, iface, srcIp, dstIp string, neighbors []*connectioncontext.IpNeighbor) *crossconnect.CrossConnect {
	srcNetNs := GetNetNS(fixture.k8s, fixture.sourcePod)
	dstNetNs := GetNetNS(fixture.k8s, fixture.destPod)
	xcon := CreateLocalCrossConnectRequest(id, "kernel", "kernel", iface, srcIp, dstIp, srcNetNs, dstNetNs, neighbors)
	return fixture.Dataplane.Request(xcon)
}

func (fixture *StandaloneDataplaneLocalFixture) RequestKernelConnection(id, iface, srcIp, dstIp string) *crossconnect.CrossConnect {
	return fixture.RequestKernelConnectionArp(id, iface, srcIp, dstIp, []*connectioncontext.IpNeighbor{})
}

func (fixture *StandaloneDataplaneLocalFixture) RequestDefaultKernelConnection() *crossconnect.CrossConnect {
	return fixture.RequestKernelConnectionArp("some-id", "iface", "10.30.1.1/29", "10.30.1.2/29", []*connectioncontext.IpNeighbor{
		{
			Ip:              "10.30.1.3",
			HardwareAddress: "aa:ff:aa:ff:aa:01",
		},
	})
}

func (fixture *StandaloneDataplaneLocalFixture) CloseConnection(xcon *crossconnect.CrossConnect) {
	fixture.Dataplane.CloseConnection(xcon)
}

func (fixture *StandaloneDataplaneLocalFixture) VerifyKernelConnection(xcon *crossconnect.CrossConnect) {
	srcIface := getIface(xcon.GetLocalSource())
	dstIface := getIface(xcon.GetLocalDestination())
	srcIp := unmaskIp(xcon.GetLocalSource().Context.SrcIpAddr)
	dstIp := unmaskIp(xcon.GetLocalSource().Context.DstIpAddr)
	VerifyKernelConnectionEstablished(fixture.k8s, fixture.sourcePod, srcIface, srcIp, dstIp)
	VerifyKernelConnectionEstablished(fixture.k8s, fixture.destPod, dstIface, dstIp, srcIp)
}

func (fixture *StandaloneDataplaneLocalFixture) VerifyKernelConnectionClosed(xcon *crossconnect.CrossConnect) {
	srcIface := getIface(xcon.GetLocalSource())
	dstIface := getIface(xcon.GetLocalDestination())
	VerifyKernelConnectionClosed(fixture.k8s, fixture.sourcePod, srcIface)
	VerifyKernelConnectionClosed(fixture.k8s, fixture.destPod, dstIface)
}
