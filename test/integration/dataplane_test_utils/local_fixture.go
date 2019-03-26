package dataplane_test_utils

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"strings"
	"testing"
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
	test      *testing.T
}

func CreateLocalFixture(test *testing.T, timeout time.Duration) *StandaloneDataplaneLocalFixture {
	fixture := &StandaloneDataplaneLocalFixture{
		timeout: timeout,
		test:    test,
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
	return fixture.RequestKernelConnection("some-id", "iface", "10.30.1.1/30", "10.30.1.2/30")
}

func (fixture *StandaloneDataplaneLocalFixture) CloseConnection(xcon *crossconnect.CrossConnect) {
	fixture.Dataplane.CloseConnection(xcon)
}

func (fixture *StandaloneDataplaneLocalFixture) VerifyKernelConnection(xcon *crossconnect.CrossConnect) {
	failures := InterceptGomegaFailures(func() {
		srcIface := getIface(xcon.GetLocalSource())
		dstIface := getIface(xcon.GetLocalDestination())
		srcIp := unmaskIp(xcon.GetLocalSource().Context.SrcIpAddr)
		dstIp := unmaskIp(xcon.GetLocalSource().Context.DstIpAddr)
		VerifyKernelConnectionEstablished(fixture.k8s, fixture.sourcePod, srcIface, srcIp, dstIp)
		VerifyKernelConnectionEstablished(fixture.k8s, fixture.destPod, dstIface, dstIp, srcIp)
	})

	fixture.handleFailures(failures)
}

func (fixture *StandaloneDataplaneLocalFixture) VerifyKernelConnectionClosed(xcon *crossconnect.CrossConnect) {
	failures := InterceptGomegaFailures(func() {
		srcIface := getIface(xcon.GetLocalSource())
		dstIface := getIface(xcon.GetLocalDestination())
		VerifyKernelConnectionClosed(fixture.k8s, fixture.sourcePod, srcIface)
		VerifyKernelConnectionClosed(fixture.k8s, fixture.destPod, dstIface)
	})

	fixture.handleFailures(failures)
}

func (fixture *StandaloneDataplaneLocalFixture) handleFailures(failures []string) {
	if len(failures) > 0 {
		for _, failure := range failures {
			logrus.Errorf("test failure: %s\n", failure)
		}
		// print logs
		fixture.PrintLogs(fixture.Dataplane.Pod())
		// print diagnostics
		fixture.PrintCommand(fixture.Dataplane.Pod(), "vppctl", "sh", "int")
		fixture.PrintCommand(fixture.Dataplane.Pod(), "ip", "addr")
		fixture.PrintCommand(fixture.sourcePod, "ip", "addr")
		fixture.PrintCommand(fixture.destPod, "ip", "addr")
		// fail test
		fixture.test.FailNow()
	}
}

func (fixture *StandaloneDataplaneLocalFixture) PrintLogs(pod *v1.Pod) {
	logs, _ := fixture.k8s.GetLogs(pod, firstContainer(pod))
	logrus.Errorf("Logs of '%s':\n%s\n", pod.Name, logs)
}

func (fixture *StandaloneDataplaneLocalFixture) PrintCommand(pod *v1.Pod, command ...string) {
	out, _, _ := fixture.k8s.Exec(pod, firstContainer(pod), command...)
	logrus.Errorf("Output of '%s' on '%s':\n%s\n", strings.Join(command, " "), pod.Name, out)
}
