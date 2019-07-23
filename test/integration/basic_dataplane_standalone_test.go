// +build basic

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"net"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	dataplaneapi "github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

const (
	dataplanePort       = 9500
	dataplaneSocketType = "tcp"
	dataplanePortName   = "dataplane"
	dataplaneProtocol   = "TCP"

	srcIp       = "10.30.1.1"
	dstIp       = "10.30.1.2"
	srcIpMasked = srcIp + "/30"
	dstIpMasked = dstIp + "/30"
)

func TestDataplaneCrossConnectBasic(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := createFixture(t, defaultTimeout)
	defer fixture.cleanup()

	conn := fixture.requestDefaultKernelConnection()
	fixture.verifyKernelConnection(conn)
}

func TestDataplaneCrossConnectMultiple(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := createFixture(t, defaultTimeout)
	defer fixture.cleanup()

	first := fixture.requestKernelConnection("id-1", "if1", "10.30.1.1/30", "10.30.1.2/30")
	second := fixture.requestKernelConnection("id-2", "if2", "10.30.2.1/30", "10.30.2.2/30")
	fixture.verifyKernelConnection(first)
	fixture.verifyKernelConnection(second)
}

func TestDataplaneCrossConnectUpdate(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := createFixture(t, defaultTimeout)
	defer fixture.cleanup()

	const someId = "0"

	orig := fixture.requestKernelConnection(someId, "if1", "10.30.1.1/30", "10.30.1.2/30")
	fixture.verifyKernelConnection(orig)

	updated := fixture.requestKernelConnection(someId, "if2", "10.30.2.1/30", "10.30.2.2/30")
	fixture.verifyKernelConnection(updated)
	fixture.verifyKernelConnectionClosed(orig)
}

func TestDataplaneCrossConnectReconnect(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	fixture := createFixture(t, defaultTimeout)
	defer fixture.cleanup()

	conn := fixture.requestDefaultKernelConnection()
	fixture.verifyKernelConnection(conn)

	fixture.closeConnection(conn)
	fixture.verifyKernelConnectionClosed(conn)

	conn = fixture.request(conn) // request the same connection
	fixture.verifyKernelConnection(conn)
}

// A standaloneDataplaneFixture represents minimalist test configuration
// with just a dataplane pod and two peer pods (source and destination)
// deployed on a single node.
type standaloneDataplaneFixture struct {
	timeout         time.Duration
	k8s             *kubetest.K8s
	node            *v1.Node
	dataplanePod    *v1.Pod
	sourcePod       *v1.Pod
	destPod         *v1.Pod
	forwarding      *kubetest.PortForward
	dataplaneClient dataplaneapi.DataplaneClient
	test            *testing.T
}

func (fixture *standaloneDataplaneFixture) cleanup() {
	fixture.forwarding.Stop()
	// Let's delete source/destPod without gracetimeout
	fixture.k8s.DeletePodsForce(fixture.sourcePod, fixture.destPod)
	fixture.k8s.Cleanup()
}

func createFixture(test *testing.T, timeout time.Duration) *standaloneDataplaneFixture {
	fixture := &standaloneDataplaneFixture{
		timeout: timeout,
		test:    test,
	}

	k8s, err := kubetest.NewK8s(true)
	Expect(err).To(BeNil())

	// prepare node
	nodes := k8s.GetNodesWait(1, timeout)
	Expect(len(nodes) >= 1).To(Equal(true), "At least one kubernetes node is required for this test")

	fixture.k8s = k8s
	fixture.node = &nodes[0]
	fixture.dataplanePod = fixture.k8s.CreatePod(dataplanePodTemplate(k8s.GetForwardingPlane(), fixture.node))
	fixture.k8s.WaitLogsContains(fixture.dataplanePod, fixture.dataplanePod.Spec.Containers[0].Name, "Serve starting...", timeout)

	// deploy source and destination pods
	fixture.sourcePod = k8s.CreatePod(pods.AlpinePod(fmt.Sprintf("source-pod-%s", fixture.node.Name), fixture.node))
	fixture.destPod = k8s.CreatePod(pods.AlpinePod(fmt.Sprintf("dest-pod-%s", fixture.node.Name), fixture.node))

	// forward dataplane port
	fixture.forwardDataplanePort(dataplanePort)

	// connect to dataplane
	fixture.connectDataplane()

	return fixture
}

func (fixture *standaloneDataplaneFixture) forwardDataplanePort(port int) {
	fwd, err := fixture.k8s.NewPortForwarder(fixture.dataplanePod, port)
	Expect(err).To(BeNil())

	err = fwd.Start()
	Expect(err).To(BeNil())
	logrus.Infof("Forwarded port: pod=%s, remote=%d local=%d\n", fixture.dataplanePod.Name, port, fwd.ListenPort)
	fixture.forwarding = fwd
}

func (fixture *standaloneDataplaneFixture) connectDataplane() {
	dataplaneConn, err := tools.DialTimeout(localPort(dataplaneSocketType, fixture.forwarding.ListenPort), 5*time.Second)
	Expect(err).To(BeNil())
	fixture.dataplaneClient = dataplaneapi.NewDataplaneClient(dataplaneConn)
}

func (fixture *standaloneDataplaneFixture) requestCrossConnect(id, srcMech, dstMech, iface, srcIp, dstIp string) *crossconnect.CrossConnect {
	req := fixture.createCrossConnectRequest(id, srcMech, dstMech, iface, srcIp, dstIp)
	return fixture.request(req)
}

func (fixture *standaloneDataplaneFixture) request(req *crossconnect.CrossConnect) *crossconnect.CrossConnect {
	ctx, _ := context.WithTimeout(context.Background(), fixture.timeout)
	conn, err := fixture.dataplaneClient.Request(ctx, req)
	Expect(err).To(BeNil())
	return conn
}

func (fixture *standaloneDataplaneFixture) createCrossConnectRequest(id, srcMech, dstMech, iface, srcIp, dstIp string) *crossconnect.CrossConnect {
	conn := &crossconnect.CrossConnect{
		Id:      id,
		Payload: "IP",
	}

	conn.Source = &crossconnect.CrossConnect_LocalSource{
		LocalSource: fixture.createConnection(id+"-src", srcMech, iface+"_src", srcIp, dstIp, fixture.sourcePod),
	}

	conn.Destination = &crossconnect.CrossConnect_LocalDestination{
		LocalDestination: fixture.createConnection(id+"-dst", dstMech, iface+"_dst", srcIp, dstIp, fixture.destPod),
	}

	return conn
}

func (fixture *standaloneDataplaneFixture) createConnection(id, mech, iface, srcIp, dstIp string, pod *v1.Pod) *connection.Connection {
	mechanism := &connection.Mechanism{
		Type: common.MechanismFromString(mech),
		Parameters: map[string]string{
			connection.InterfaceNameKey:        iface,
			connection.InterfaceDescriptionKey: "Some description",
			connection.SocketFilename:          path.Join(iface, connection.MemifSocket),
			connection.NetNsInodeKey:           fixture.getNetNS(pod),
		},
	}
	err := mechanism.IsValid()
	Expect(err).To(BeNil())

	return &connection.Connection{
		Id:             id,
		NetworkService: "some-network-service",
		Mechanism:      mechanism,
		Context: &connectioncontext.ConnectionContext{
			IpContext: &connectioncontext.IPContext{
				SrcIpAddr: srcIp,
				DstIpAddr: dstIp,
			},
		},
	}
}

func (fixture *standaloneDataplaneFixture) getNetNS(pod *v1.Pod) string {
	container := pod.Spec.Containers[0].Name
	link, _, err := fixture.k8s.Exec(pod, container, "readlink", "/proc/self/ns/net")
	Expect(err).To(BeNil())

	pattern := regexp.MustCompile("net:\\[(.*)\\]")
	matches := pattern.FindStringSubmatch(link)
	Expect(len(matches) >= 1).To(BeTrue())

	return matches[1]
}

func (fixture *standaloneDataplaneFixture) requestKernelConnection(id, iface, srcIp, dstIp string) *crossconnect.CrossConnect {
	return fixture.requestCrossConnect(id, "kernel", "kernel", iface, srcIp, dstIp)
}

func (fixture *standaloneDataplaneFixture) requestDefaultKernelConnection() *crossconnect.CrossConnect {
	return fixture.requestKernelConnection("0", "iface", srcIpMasked, dstIpMasked)
}

func (fixture *standaloneDataplaneFixture) verifyKernelConnection(xcon *crossconnect.CrossConnect) {
	failures := InterceptGomegaFailures(func() {
		srcIface := getIface(xcon.GetLocalSource())
		dstIface := getIface(xcon.GetLocalDestination())
		srcIp := unmaskIp(xcon.GetLocalSource().Context.IpContext.SrcIpAddr)
		dstIp := unmaskIp(xcon.GetLocalDestination().Context.IpContext.DstIpAddr)

		out, _, err := fixture.k8s.Exec(fixture.sourcePod, fixture.sourcePod.Spec.Containers[0].Name, "ifconfig", srcIface)
		Expect(err).To(BeNil())
		Expect(strings.Contains(out, fmt.Sprintf("inet addr:%s", srcIp))).To(BeTrue())

		logrus.Infof("Source interface:\n%s", out)

		out, _, err = fixture.k8s.Exec(fixture.destPod, fixture.destPod.Spec.Containers[0].Name, "ifconfig", dstIface)
		Expect(err).To(BeNil())
		Expect(strings.Contains(out, fmt.Sprintf("inet addr:%s", dstIp))).To(BeTrue())

		logrus.Infof("Destination interface:\n%s", out)

		out, _, err = fixture.k8s.Exec(fixture.sourcePod, fixture.sourcePod.Spec.Containers[0].Name, "ping", dstIp, "-c", "1")
		Expect(err).To(BeNil())
		Expect(strings.Contains(out, "0% packet loss")).To(BeTrue())
	})

	fixture.handleFailures(failures)
}

func (fixture *standaloneDataplaneFixture) handleFailures(failures []string) {
	if len(failures) > 0 {
		for _, failure := range failures {
			logrus.Errorf("test failure: %s\n", failure)
		}
		fixture.printLogs(fixture.dataplanePod)
		fixture.printLogs(fixture.sourcePod)
		fixture.printLogs(fixture.destPod)
		fixture.test.Fail()
		kubetest.ShowLogs(fixture.k8s, fixture.test)
	}
}

func (fixture *standaloneDataplaneFixture) printLogs(pod *v1.Pod) {
	logs, _ := fixture.k8s.GetLogs(pod, firstContainer(pod))
	logrus.Errorf("=================================\nLogs of '%s' pod:\n%s\n", pod.Name, logs)
}

func (fixture *standaloneDataplaneFixture) verifyKernelConnectionClosed(xcon *crossconnect.CrossConnect) {
	failures := InterceptGomegaFailures(func() {
		srcIface := getIface(xcon.GetLocalSource())
		dstIface := getIface(xcon.GetLocalDestination())

		out, _, err := fixture.k8s.Exec(fixture.sourcePod, fixture.sourcePod.Spec.Containers[0].Name, "ip", "a")
		Expect(err).To(BeNil())
		Expect(strings.Contains(out, srcIface)).To(BeFalse())

		logrus.Infof("Source interfaces:\n%s", out)

		out, _, err = fixture.k8s.Exec(fixture.destPod, fixture.destPod.Spec.Containers[0].Name, "ip", "a")
		Expect(err).To(BeNil())
		Expect(strings.Contains(out, dstIface)).To(BeFalse())

		logrus.Infof("Destination interfaces:\n%s", out)
	})

	fixture.handleFailures(failures)
}

func (fixture *standaloneDataplaneFixture) closeConnection(conn *crossconnect.CrossConnect) {
	ctx, _ := context.WithTimeout(context.Background(), fixture.timeout)
	_, err := fixture.dataplaneClient.Close(ctx, conn)
	Expect(err).To(BeNil())
}

func unmaskIp(maskedIp string) string {
	return strings.Split(maskedIp, "/")[0]
}

func maskIp(ip, mask string) string {
	return fmt.Sprintf("%s/%s", ip, mask)
}

func getIface(conn *connection.Connection) string {
	return conn.Mechanism.Parameters[connection.InterfaceNameKey]
}

func localPort(network string, port int) net.Addr {
	return &net.UnixAddr{
		Net:  network,
		Name: fmt.Sprintf("localhost:%d", port),
	}
}

func dataplanePodTemplate(plane string, node *v1.Node) *v1.Pod {
	dataplaneName := fmt.Sprintf("nsmd-dataplane-%s", node.Name)
	dataplane := pods.ForwardingPlane(dataplaneName, node, plane)
	setupEnvVariables(dataplane, map[string]string{
		"DATAPLANE_SOCKET_TYPE": dataplaneSocketType,
		"DATAPLANE_SOCKET":      fmt.Sprintf("0.0.0.0:%d", dataplanePort),
	})
	exposePorts(dataplane,
		v1.ContainerPort{
			ContainerPort: dataplanePort,
			Name:          dataplanePortName,
			Protocol:      dataplaneProtocol,
		},
		v1.ContainerPort{
			ContainerPort: 40000,
			Name:          "debug",
			Protocol:      dataplaneProtocol,
		})
	dataplane.ObjectMeta.Labels = map[string]string{"run": "dataplane"}
	return dataplane
}

func setupEnvVariables(dataplane *v1.Pod, env map[string]string) {
	vpp := &dataplane.Spec.Containers[0]

	environment := vpp.Env
	for key, value := range env {
		environment = append(environment, v1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	vpp.Env = environment
}

func exposePorts(dataplane *v1.Pod, ports ...v1.ContainerPort) {
	vpp := &dataplane.Spec.Containers[0]
	vpp.Ports = append(vpp.Ports, ports...)
}

func firstContainer(pod *v1.Pod) string {
	return pod.Spec.Containers[0].Name
}
