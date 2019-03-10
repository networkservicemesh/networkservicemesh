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

	srcIp     = "10.30.1.1"
	dstIp     = "10.30.1.2"
	srcIpMask = srcIp + "/30"
	dstIpMask = dstIp + "/30"
)

func TestDataplaneCrossConnectMemMem(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneCrossConnect("mem", "mem", defaultTimeout)
}

func TestDataplaneCrossConnectKernelKernel(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneCrossConnect("kernel", "kernel", defaultTimeout)
}

func TestDataplaneCrossConnectKernelMem(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testDataplaneCrossConnect("kernel", "mem", defaultTimeout)
}

func testDataplaneCrossConnect(sourceMech, destMech string, timeout time.Duration) {
	// deploy dataplane
	k8s, dataplane, node := deployVppDataplane(timeout)
	defer k8s.Cleanup()

	// deploy source and destination pods
	source := k8s.CreatePod(pods.AlpinePod(fmt.Sprintf("source-pod-%s", node.Name), node))
	dest := k8s.CreatePod(pods.AlpinePod(fmt.Sprintf("dest-pod-%s", node.Name), node))

	// bind dataplane to local port
	forwarding := forwardPort(k8s, dataplane, dataplanePort)
	defer forwarding.Stop()

	// connect to dataplane
	dataplaneClient := connectToDataplane(forwarding.ListenPort)

	// create cross-connection
	connect := newCrossConnectRequest(k8s, source, dest, sourceMech, destMech)

	// request cross-connection
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	connect, err := dataplaneClient.Request(ctx, connect)
	Expect(err).To(BeNil())

	// verify connection is established
	verifyConnectionEstablished(k8s, source, dest, connect)
}

func verifyConnectionEstablished(k8s *kube_testing.K8s, source, destination *v1.Pod, xcon *crossconnect.CrossConnect) {
	srcIface := getIface(xcon.GetLocalSource())
	out, _, err := k8s.Exec(source, source.Spec.Containers[0].Name, "ifconfig", srcIface)
	Expect(err).To(BeNil())
	Expect(strings.Contains(out, fmt.Sprintf("inet addr:%s", srcIp))).To(BeTrue())

	logrus.Infof("Source interface:\n%s", out)

	dstIface := getIface(xcon.GetLocalDestination())
	out, _, err = k8s.Exec(destination, destination.Spec.Containers[0].Name, "ifconfig", dstIface)
	Expect(err).To(BeNil())
	Expect(strings.Contains(out, fmt.Sprintf("inet addr:%s", dstIp))).To(BeTrue())

	logrus.Infof("Destination interface:\n%s", out)

	out, _, err = k8s.Exec(source, source.Spec.Containers[0].Name, "ping", dstIp, "-c", "1")
	Expect(err).To(BeNil())
	Expect(strings.Contains(out, "0% packet loss")).To(BeTrue())
}

func getIface(conn *connection.Connection) string {
	return conn.Mechanism.Parameters[connection.InterfaceNameKey]
}

func forwardPort(k8s *kube_testing.K8s, pod *v1.Pod, port int) *kube_testing.PortForward {
	forwarding, err := k8s.NewPortForwarder(pod, port)
	Expect(err).To(BeNil())

	err = forwarding.Start()
	Expect(err).To(BeNil())
	logrus.Infof("Forwarded port: pod=%s, remote port=%d local port=%d\n", pod.Name, port, forwarding.ListenPort)
	return forwarding
}

func newCrossConnectRequest(k8s *kube_testing.K8s, source, dest *v1.Pod, sourceMechanism, destMechanism string) *crossconnect.CrossConnect {
	conn := &crossconnect.CrossConnect{
		Id:      "some-id",
		Payload: "IP",
	}

	conn.Source = &crossconnect.CrossConnect_LocalSource{
		LocalSource: newConnection(k8s, "conn-id-1", "ns-service-1", destMechanism, "iface_src", source),
	}

	conn.GetLocalSource().Context.SrcIpAddr = srcIpMask
	conn.GetLocalSource().Context.DstIpAddr = dstIpMask

	conn.Destination = &crossconnect.CrossConnect_LocalDestination{
		LocalDestination: newConnection(k8s, "conn-id-2", "ns-service-2", destMechanism, "iface_dst", dest),
	}

	conn.GetLocalDestination().Context.SrcIpAddr = srcIpMask
	conn.GetLocalDestination().Context.DstIpAddr = dstIpMask

	return conn
}

func newConnection(k8s *kube_testing.K8s, id, netService, mech, iface string, pod *v1.Pod) *connection.Connection {
	mechanismType := common.MechanismFromString(mech)
	mechanism, err := connection.NewMechanism(mechanismType, iface, "Primary interface")
	Expect(err).To(BeNil())

	mechanism.Parameters[connection.NetNsInodeKey] = getNetworkNamespace(k8s, pod)

	return &connection.Connection{
		Id:             id,
		NetworkService: netService,
		Mechanism:      mechanism,

		Context: &connectioncontext.ConnectionContext{},
	}

}

func getNetworkNamespace(k8s *kube_testing.K8s, pod *v1.Pod) string {
	container := pod.Spec.Containers[0].Name
	link, _, err := k8s.Exec(pod, container, "readlink", "/proc/self/ns/net")
	Expect(err).To(BeNil())

	pattern := regexp.MustCompile("net:\\[(.*)\\]")
	matches := pattern.FindStringSubmatch(link)
	Expect(len(matches) >= 1).To(BeTrue())

	return matches[1]
}

func connectToDataplane(port int) dataplaneapi.DataplaneClient {
	dataplaneConn, err := tools.SocketOperationCheck(localPort(dataplaneSocketType, port))
	Expect(err).To(BeNil())

	dataplaneClient := dataplaneapi.NewDataplaneClient(dataplaneConn)
	return dataplaneClient
}

func localPort(network string, port int) net.Addr {
	return &net.UnixAddr{
		Net:  network,
		Name: fmt.Sprintf("localhost:%d", port),
	}
}

func deployVppDataplane(timeout time.Duration) (*kube_testing.K8s, *v1.Pod, *v1.Node) {
	k8s, err := kube_testing.NewK8s()
	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse", "jaeger")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	// prepare node
	nodes := k8s.GetNodesWait(1, timeout)
	Expect(len(nodes) >= 1).To(Equal(true), "At least one kubernetes node is required for this test")
	node := &nodes[0]

	// deploy dataplane pod
	dataplane := k8s.CreatePod(DataplanePodTemplate(node))

	return k8s, dataplane, node
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
