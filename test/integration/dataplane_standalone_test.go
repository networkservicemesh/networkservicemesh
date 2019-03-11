// +build basic

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	dataplaneapi "github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"
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

	fixture := createFixture(defaultTimeout)
	defer fixture.cleanup()

	conn := fixture.requestDefaultKernelConnection()
	fixture.verifyKernelConnection(conn)
}

// A standaloneDataplaneFixture represents minimalist test configuration
// with just a dataplane pod and two peer pods (source and destination)
// deployed on a single node.
type standaloneDataplaneFixture struct {
	timeout         time.Duration
	k8s             *kube_testing.K8s
	node            *v1.Node
	dataplanePod    *v1.Pod
	sourcePod       *v1.Pod
	destPod         *v1.Pod
	forwarding      *kube_testing.PortForward
	dataplaneClient dataplaneapi.DataplaneClient
}

func (fixture *standaloneDataplaneFixture) cleanup() {
	fixture.forwarding.Stop()
	fixture.k8s.Cleanup()
}

func createFixture(timeout time.Duration) *standaloneDataplaneFixture {
	fixture := &standaloneDataplaneFixture{
		timeout: timeout,
	}

	k8s, err := kube_testing.NewK8s()
	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse", "jaeger")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	// prepare node
	nodes := k8s.GetNodesWait(1, timeout)
	Expect(len(nodes) >= 1).To(Equal(true), "At least one kubernetes node is required for this test")

	fixture.k8s = k8s
	fixture.node = &nodes[0]
	fixture.dataplanePod = fixture.k8s.CreatePod(DataplanePodTemplate(fixture.node))

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
	dataplaneConn, err := tools.SocketOperationCheck(localPort(dataplaneSocketType, fixture.forwarding.ListenPort))
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
	mechanismType := common.MechanismFromString(mech)
	mechanism, err := connection.NewMechanism(mechanismType, iface, "")
	Expect(err).To(BeNil())

	mechanism.Parameters[connection.NetNsInodeKey] = fixture.getNetNS(pod)

	return &connection.Connection{
		Id:             id,
		NetworkService: "some-network-service",
		Mechanism:      mechanism,
		Context: &connectioncontext.ConnectionContext{
			SrcIpAddr: srcIp,
			DstIpAddr: dstIp,
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

func (fixture *standaloneDataplaneFixture) requestDefaultKernelConnection() *crossconnect.CrossConnect {
	return fixture.requestCrossConnect("some-id", "kernel", "kernel", "iface", srcIpMasked, dstIpMasked)
}

func (fixture *standaloneDataplaneFixture) verifyKernelConnection(xcon *crossconnect.CrossConnect) {
	srcIface := getIface(xcon.GetLocalSource())
	dstIface := getIface(xcon.GetLocalDestination())
	srcIp := unmaskIp(xcon.GetLocalSource().Context.SrcIpAddr)
	dstIp := unmaskIp(xcon.GetLocalDestination().Context.DstIpAddr)

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

func DataplanePodTemplate(node *v1.Node) *v1.Pod {
	dataplaneName := fmt.Sprintf("nsmd-dataplane-%s", node.Name)
	dataplane := pods.VPPDataplanePod(dataplaneName, node)
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
