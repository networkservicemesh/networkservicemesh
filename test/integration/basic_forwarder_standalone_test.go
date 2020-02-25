// +build basic

package integration

import (
	"context"
	"fmt"
	"net"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	forwarderapi "github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

const (
	forwarderPort       = 9500
	forwarderSocketType = "tcp"
	forwarderPortName   = "forwarder"
	forwarderProtocol   = "TCP"

	srcIp       = "10.30.1.1"
	dstIp       = "10.30.1.2"
	srcIpMasked = srcIp + "/30"
	dstIpMasked = dstIp + "/30"
)

var wt *WithT

func TestForwarderCrossConnectBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	wt = NewWithT(t)

	fixture := createFixture(t, defaultTimeout)
	defer fixture.cleanup()

	conn := fixture.requestDefaultKernelConnection()
	fixture.verifyKernelConnection(conn)
}

func TestForwarderCrossConnectMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	wt = NewWithT(t)

	fixture := createFixture(t, defaultTimeout)
	defer fixture.cleanup()

	first := fixture.requestKernelConnection("id-1", "if1", "10.30.1.1/30", "10.30.1.2/30")
	second := fixture.requestKernelConnection("id-2", "if2", "10.30.2.1/30", "10.30.2.2/30")
	fixture.verifyKernelConnection(first)
	fixture.verifyKernelConnection(second)
}

func TestForwarderCrossConnectUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	wt = NewWithT(t)

	fixture := createFixture(t, defaultTimeout)
	defer fixture.cleanup()

	const someId = "0"

	orig := fixture.requestKernelConnection(someId, "if1", "10.30.1.1/30", "10.30.1.2/30")
	fixture.verifyKernelConnection(orig)

	updated := fixture.requestKernelConnection(someId, "if2", "10.30.2.1/30", "10.30.2.2/30")
	fixture.verifyKernelConnection(updated)
	fixture.verifyKernelConnectionClosed(orig)
}

func TestForwarderCrossConnectReconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	wt = NewWithT(t)

	fixture := createFixture(t, defaultTimeout)
	defer fixture.cleanup()

	conn := fixture.requestDefaultKernelConnection()
	fixture.verifyKernelConnection(conn)

	fixture.closeConnection(conn)
	fixture.verifyKernelConnectionClosed(conn)

	conn = fixture.request(conn) // request the same connection
	fixture.verifyKernelConnection(conn)
}

// A standaloneForwarderFixture represents minimalist test configuration
// with just a forwarder pod and two peer pods (source and destination)
// deployed on a single node.
type standaloneForwarderFixture struct {
	timeout         time.Duration
	k8s             *kubetest.K8s
	node            *v1.Node
	forwarderPod    *v1.Pod
	sourcePod       *v1.Pod
	destPod         *v1.Pod
	forwarding      *kubetest.PortForward
	forwarderClient forwarderapi.ForwarderClient
	test            *testing.T
}

func (fixture *standaloneForwarderFixture) cleanup() {
	fixture.forwarding.Stop()
	// Let's delete source/destPod without gracetimeout
	fixture.k8s.DeletePodsForce(fixture.sourcePod, fixture.destPod)
	fixture.k8s.Cleanup()
}

func createFixture(test *testing.T, timeout time.Duration) *standaloneForwarderFixture {
	fixture := &standaloneForwarderFixture{
		timeout: timeout,
		test:    test,
	}

	k8s, err := kubetest.NewK8s(wt, true)
	wt.Expect(err).To(BeNil())

	// prepare node
	nodes := k8s.GetNodesWait(1, timeout)
	wt.Expect(len(nodes) >= 1).To(Equal(true), "At least one kubernetes node is required for this test")

	fixture.k8s = k8s
	fixture.node = &nodes[0]
	fixture.forwarderPod = fixture.k8s.CreatePod(forwarderPodTemplate(k8s.GetForwardingPlane(), fixture.node))
	fixture.k8s.WaitLogsContains(fixture.forwarderPod, fixture.forwarderPod.Spec.Containers[0].Name, "Serve starting...", timeout)

	// deploy source and destination pods
	fixture.sourcePod = k8s.CreatePod(pods.AlpinePod(fmt.Sprintf("source-pod-%s", fixture.node.Name), fixture.node))
	fixture.destPod = k8s.CreatePod(pods.AlpinePod(fmt.Sprintf("dest-pod-%s", fixture.node.Name), fixture.node))

	// forward forwarder port
	fixture.forwardForwarderPort(forwarderPort)

	// connect to forwarder
	fixture.connectForwarder()

	return fixture
}

func (fixture *standaloneForwarderFixture) forwardForwarderPort(port int) {
	fwd, err := fixture.k8s.NewPortForwarder(fixture.forwarderPod, port)
	wt.Expect(err).To(BeNil())

	err = fwd.Start()
	wt.Expect(err).To(BeNil())
	logrus.Infof("Forwarded port: pod=%s, remote=%d local=%d\n", fixture.forwarderPod.Name, port, fwd.ListenPort)
	fixture.forwarding = fwd
}

func (fixture *standaloneForwarderFixture) connectForwarder() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	forwarderConn, err := tools.DialContext(ctx, localPort(forwarderSocketType, fixture.forwarding.ListenPort))
	wt.Expect(err).To(BeNil())
	fixture.forwarderClient = forwarderapi.NewForwarderClient(forwarderConn)
}

func (fixture *standaloneForwarderFixture) requestCrossConnect(id, srcMech, dstMech, iface, srcIp, dstIp string) *crossconnect.CrossConnect {
	req := fixture.createCrossConnectRequest(id, srcMech, dstMech, iface, srcIp, dstIp)
	return fixture.request(req)
}

func (fixture *standaloneForwarderFixture) request(req *crossconnect.CrossConnect) *crossconnect.CrossConnect {
	ctx, _ := context.WithTimeout(context.Background(), fixture.timeout)
	conn, err := fixture.forwarderClient.Request(ctx, req)
	wt.Expect(err).To(BeNil())
	return conn
}

func (fixture *standaloneForwarderFixture) createCrossConnectRequest(id, srcMech, dstMech, iface, srcIp, dstIp string) *crossconnect.CrossConnect {
	conn := &crossconnect.CrossConnect{
		Id:      id,
		Payload: "IP",
	}

	conn.Source = fixture.createConnection(id+"-src", srcMech, iface+"_src", srcIp, dstIp, fixture.sourcePod)

	conn.Destination = fixture.createConnection(id+"-dst", dstMech, iface+"_dst", srcIp, dstIp, fixture.destPod)

	return conn
}

func (fixture *standaloneForwarderFixture) createConnection(id, mech, iface, srcIp, dstIp string, pod *v1.Pod) *networkservice.Connection {
	mechanism := &networkservice.Mechanism{
		Type: mech,
		Parameters: map[string]string{
			common.InterfaceNameKey:        iface,
			common.InterfaceDescriptionKey: "Some description",
			memif.SocketFilename:           path.Join(iface, memif.MemifSocket),
			common.NetNSInodeKey:           fixture.getNetNS(pod),
		},
	}
	err := mechanism.IsValid()
	wt.Expect(err).To(BeNil())

	return &networkservice.Connection{
		Id:             id,
		NetworkService: "some-network-service",
		Mechanism:      mechanism,
		Context: &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				SrcIpAddr: srcIp,
				DstIpAddr: dstIp,
			},
		},
	}
}

func (fixture *standaloneForwarderFixture) getNetNS(pod *v1.Pod) string {
	container := pod.Spec.Containers[0].Name
	link, _, err := fixture.k8s.Exec(pod, container, "readlink", "/proc/self/ns/net")
	wt.Expect(err).To(BeNil())

	pattern := regexp.MustCompile("net:\\[(.*)\\]")
	matches := pattern.FindStringSubmatch(link)
	wt.Expect(len(matches) >= 1).To(BeTrue())

	return matches[1]
}

func (fixture *standaloneForwarderFixture) requestKernelConnection(id, iface, srcIp, dstIp string) *crossconnect.CrossConnect {
	return fixture.requestCrossConnect(id, kernel.MECHANISM, kernel.MECHANISM, iface, srcIp, dstIp)
}

func (fixture *standaloneForwarderFixture) requestDefaultKernelConnection() *crossconnect.CrossConnect {
	return fixture.requestKernelConnection("0", "iface", srcIpMasked, dstIpMasked)
}

func (fixture *standaloneForwarderFixture) verifyKernelConnection(xcon *crossconnect.CrossConnect) {
	srcIface := getIface(xcon.GetSource())
	dstIface := getIface(xcon.GetDestination())
	srcIp := unmaskIp(xcon.GetSource().GetContext().GetIpContext().GetSrcIpAddr())
	dstIp := unmaskIp(xcon.GetDestination().GetContext().GetIpContext().GetDstIpAddr())

	out, _, err := fixture.k8s.Exec(fixture.sourcePod, fixture.sourcePod.Spec.Containers[0].Name, "ifconfig", srcIface)
	wt.Expect(err).To(BeNil())
	wt.Expect(strings.Contains(out, fmt.Sprintf("inet addr:%s", srcIp))).To(BeTrue())

	logrus.Infof("Source interface:\n%s", out)

	out, _, err = fixture.k8s.Exec(fixture.destPod, fixture.destPod.Spec.Containers[0].Name, "ifconfig", dstIface)
	wt.Expect(err).To(BeNil())
	wt.Expect(strings.Contains(out, fmt.Sprintf("inet addr:%s", dstIp))).To(BeTrue())

	logrus.Infof("Destination interface:\n%s", out)

	out, _, err = fixture.k8s.Exec(fixture.sourcePod, fixture.sourcePod.Spec.Containers[0].Name, "ping", dstIp, "-c", "1")
	wt.Expect(err).To(BeNil())
	wt.Expect(strings.Contains(out, "0% packet loss")).To(BeTrue())
}

func (fixture *standaloneForwarderFixture) handleFailures(failures []string) {
	if len(failures) > 0 {
		for _, failure := range failures {
			logrus.Errorf("test failure: %s\n", failure)
		}
		fixture.test.Fail()
		defer fixture.k8s.SaveTestArtifacts(fixture.test)
	}
}

func (fixture *standaloneForwarderFixture) printLogs(pod *v1.Pod) {
	logs, _ := fixture.k8s.GetLogs(pod, firstContainer(pod))
	logrus.Errorf("=================================\nLogs of '%s' pod:\n%s\n", pod.Name, logs)
}

func (fixture *standaloneForwarderFixture) verifyKernelConnectionClosed(xcon *crossconnect.CrossConnect) {
	srcIface := getIface(xcon.GetSource())
	dstIface := getIface(xcon.GetDestination())

	out, _, err := fixture.k8s.Exec(fixture.sourcePod, fixture.sourcePod.Spec.Containers[0].Name, "ip", "a")
	wt.Expect(err).To(BeNil())
	wt.Expect(strings.Contains(out, srcIface)).To(BeFalse())

	logrus.Infof("Source interfaces:\n%s", out)

	out, _, err = fixture.k8s.Exec(fixture.destPod, fixture.destPod.Spec.Containers[0].Name, "ip", "a")
	wt.Expect(err).To(BeNil())
	wt.Expect(strings.Contains(out, dstIface)).To(BeFalse())

	logrus.Infof("Destination interfaces:\n%s", out)
}

func (fixture *standaloneForwarderFixture) closeConnection(conn *crossconnect.CrossConnect) {
	ctx, _ := context.WithTimeout(context.Background(), fixture.timeout)
	_, err := fixture.forwarderClient.Close(ctx, conn)
	wt.Expect(err).To(BeNil())
}

func unmaskIp(maskedIp string) string {
	return strings.Split(maskedIp, "/")[0]
}

func maskIp(ip, mask string) string {
	return fmt.Sprintf("%s/%s", ip, mask)
}

func getIface(conn *networkservice.Connection) string {
	return conn.Mechanism.Parameters[common.InterfaceNameKey]
}

func localPort(network string, port int) net.Addr {
	return &net.UnixAddr{
		Net:  network,
		Name: fmt.Sprintf("localhost:%d", port),
	}
}

func forwarderPodTemplate(plane string, node *v1.Node) *v1.Pod {
	forwarderName := fmt.Sprintf("nsmd-forwarder-%s", node.Name)
	forwarder := pods.ForwardingPlane(forwarderName, node, plane)
	setupEnvVariables(forwarder, map[string]string{
		"FORWARDER_SOCKET_TYPE": forwarderSocketType,
		"FORWARDER_SOCKET":      fmt.Sprintf("0.0.0.0:%d", forwarderPort),
	})
	exposePorts(forwarder,
		v1.ContainerPort{
			ContainerPort: forwarderPort,
			Name:          forwarderPortName,
			Protocol:      forwarderProtocol,
		},
		v1.ContainerPort{
			ContainerPort: 40000,
			Name:          "debug",
			Protocol:      forwarderProtocol,
		})
	forwarder.ObjectMeta.Labels = map[string]string{"run": "forwarder"}
	return forwarder
}

func setupEnvVariables(forwarder *v1.Pod, env map[string]string) {
	vpp := &forwarder.Spec.Containers[0]

	environment := vpp.Env
	for key, value := range env {
		environment = append(environment, v1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	vpp.Env = environment
}

func exposePorts(forwarder *v1.Pod, ports ...v1.ContainerPort) {
	vpp := &forwarder.Spec.Containers[0]
	vpp.Ports = append(vpp.Ports, ports...)
}

func firstContainer(pod *v1.Pod) string {
	return pod.Spec.Containers[0].Name
}
