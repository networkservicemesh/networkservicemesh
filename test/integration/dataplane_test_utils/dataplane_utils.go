package dataplane_test_utils

import (
	"context"
	"fmt"
	connctx "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	dataplaneapi "github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"net"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	dataplanePort       = 9500
	dataplaneSocketType = "tcp"
	dataplanePortName   = "dataplane"
	dataplaneProtocol   = "TCP"
)

// A StandaloneDataplaneInstance represents a single deployed dataplane
// pod that could be programmed directly via the grpc client.
type StandaloneDataplaneInstance struct {
	timeout         time.Duration
	k8s             *kube_testing.K8s
	node            *v1.Node
	dataplanePod    *v1.Pod
	forwarding      *kube_testing.PortForward
	dataplaneClient dataplaneapi.DataplaneClient
}

func CreateDataplaneInstance(k8s *kube_testing.K8s, node *v1.Node, timeout time.Duration) *StandaloneDataplaneInstance {
	instance := &StandaloneDataplaneInstance{
		k8s:     k8s,
		timeout: timeout,
		node:    node,
	}

	instance.dataplanePod = k8s.CreatePod(dataplanePodTemplate(node))
	k8s.WaitLogsContains(instance.dataplanePod, firstContainer(instance.dataplanePod), "Serve starting...", timeout)
	instance.forwarding = forwardPort(k8s, instance.dataplanePod, dataplanePort)
	instance.dataplaneClient = connectDataplane(instance.forwarding.ListenPort)

	return instance
}

func (instance *StandaloneDataplaneInstance) Cleanup() {
	instance.forwarding.Stop()
}

func (instance *StandaloneDataplaneInstance) Request(req *crossconnect.CrossConnect) *crossconnect.CrossConnect {
	ctx, _ := context.WithTimeout(context.Background(), instance.timeout)
	conn, err := instance.dataplaneClient.Request(ctx, req)
	Expect(err).To(BeNil())
	return conn
}

func (instance *StandaloneDataplaneInstance) CloseConnection(conn *crossconnect.CrossConnect) {
	ctx, _ := context.WithTimeout(context.Background(), instance.timeout)
	_, err := instance.dataplaneClient.Close(ctx, conn)
	Expect(err).To(BeNil())
}

func (instance *StandaloneDataplaneInstance) stop() {
	instance.forwarding.Stop()
	instance.k8s.DeletePods(instance.dataplanePod)
}

func (instance *StandaloneDataplaneInstance) KillAndHeal() {
	instance.forwarding.Stop()
	instance.k8s.DeletePods(instance.dataplanePod)

	instance.dataplanePod = instance.k8s.CreatePod(dataplanePodTemplate(instance.node))
	instance.forwarding = forwardPort(instance.k8s, instance.dataplanePod, dataplanePort)
	instance.dataplaneClient = connectDataplane(instance.forwarding.ListenPort)
}

func (instance *StandaloneDataplaneInstance) Pod() *v1.Pod {
	return instance.dataplanePod
}

func CreateLocalCrossConnectRequest(id, srcMech, dstMech, iface, srcIp, dstIp, srcNetNsInode, dstNetNsInode string, neighbors []*connctx.IpNeighbor) *crossconnect.CrossConnect {
	return &crossconnect.CrossConnect{
		Id:      id,
		Payload: "IP",

		Source: &crossconnect.CrossConnect_LocalSource{
			LocalSource: createLocalConnection(id+"-src", srcMech, iface+"_src", srcIp, dstIp, srcNetNsInode, neighbors),
		},

		Destination: &crossconnect.CrossConnect_LocalDestination{
			LocalDestination: createLocalConnection(id+"-dst", dstMech, iface+"_dst", srcIp, dstIp, dstNetNsInode, neighbors),
		},
	}
}

func CreateRemoteXConnectRequestSrc(id, srcMech string, dstMech *remote.Mechanism, iface, srcIp, dstIp, srcNetNsInode string, neighbors []*connctx.IpNeighbor) *crossconnect.CrossConnect {
	return &crossconnect.CrossConnect{
		Id:      id,
		Payload: "IP",

		Source: &crossconnect.CrossConnect_LocalSource{
			LocalSource: createLocalConnection(id+"-src", srcMech, iface+"_src", srcIp, dstIp, srcNetNsInode, neighbors),
		},

		Destination: &crossconnect.CrossConnect_RemoteDestination{
			RemoteDestination: createRemoteConnection(id+"-dst", srcIp, dstIp, dstMech, neighbors),
		},
	}
}

func CreateRemoteXConnectRequestDst(id string, srcMech *remote.Mechanism, dstMech, iface, srcIp, dstIp, dstNetNsInode string, neighbors []*connctx.IpNeighbor) *crossconnect.CrossConnect {
	return &crossconnect.CrossConnect{
		Id:      id,
		Payload: "IP",

		Source: &crossconnect.CrossConnect_RemoteSource{
			RemoteSource: createRemoteConnection(id+"-src", srcIp, dstIp, srcMech, neighbors),
		},

		Destination: &crossconnect.CrossConnect_LocalDestination{
			LocalDestination: createLocalConnection(id+"-dst", dstMech, iface+"_dst", srcIp, dstIp, dstNetNsInode, neighbors),
		},
	}
}

func createRemoteConnection(id, srcIp, dstIp string, mechanism *remote.Mechanism, neighbors []*connctx.IpNeighbor) *remote.Connection {
	return &remote.Connection{
		Id:             id,
		Mechanism:      mechanism,
		NetworkService: "some-network-service",

		Context: &connctx.ConnectionContext{
			SrcIpAddr:   srcIp,
			DstIpAddr:   dstIp,
			IpNeighbors: neighbors,
		},
	}
}

func vxlanMechanism(srcIp, dstIp, vni string) *remote.Mechanism {
	return &remote.Mechanism{
		Type: remote.MechanismType_VXLAN,
		Parameters: map[string]string{
			remote.VXLANSrcIP: srcIp,
			remote.VXLANDstIP: dstIp,
			remote.VXLANVNI:   vni,
		},
	}
}

func vxlanVni(vxlan *remote.Mechanism) string {
	return vxlan.Parameters[remote.VXLANVNI]
}

func createLocalConnection(id, mech, iface, srcIp, dstIp, netNsInode string, neighbors []*connctx.IpNeighbor) *connection.Connection {
	mechanism := &connection.Mechanism{
		Type: common.MechanismFromString(mech),
		Parameters: map[string]string{
			connection.InterfaceNameKey:        iface,
			connection.InterfaceDescriptionKey: "Some description",
			connection.SocketFilename:          path.Join(iface, connection.MemifSocket),
			connection.NetNsInodeKey:           netNsInode,
		},
	}
	err := mechanism.IsValid()
	Expect(err).To(BeNil())

	return &connection.Connection{
		Id:             id,
		NetworkService: "some-network-service",
		Mechanism:      mechanism,
		Context: &connctx.ConnectionContext{
			SrcIpAddr:   srcIp,
			DstIpAddr:   dstIp,
			IpNeighbors: neighbors,
		},
	}
}

func GetNetNS(k8s *kube_testing.K8s, pod *v1.Pod) string {
	container := pod.Spec.Containers[0].Name
	link, _, err := k8s.Exec(pod, container, "readlink", "/proc/self/ns/net")
	Expect(err).To(BeNil())

	pattern := regexp.MustCompile("net:\\[(.*)\\]")
	matches := pattern.FindStringSubmatch(link)
	Expect(len(matches) >= 1).To(BeTrue())

	return matches[1]
}

func VerifyKernelConnectionEstablished(k8s *kube_testing.K8s, pod *v1.Pod, iface, srcIp, dstIp string) {
	out, _, err := k8s.Exec(pod, pod.Spec.Containers[0].Name, "ifconfig", iface)
	logrus.Infof("Interface on %s:\n%s", pod.Name, out)
	Expect(err).To(BeNil())
	Expect(strings.Contains(out, fmt.Sprintf("inet addr:%s", srcIp))).To(BeTrue())

	out, _, err = k8s.Exec(pod, pod.Spec.Containers[0].Name, "ping", dstIp, "-c", "1")
	Expect(err).To(BeNil())
	Expect(strings.Contains(out, "0% packet loss")).To(BeTrue())
}

func VerifyKernelConnectionClosed(k8s *kube_testing.K8s, pod *v1.Pod, iface string) {
	out, _, err := k8s.Exec(pod, pod.Spec.Containers[0].Name, "ip", "a")
	Expect(err).To(BeNil())
	Expect(strings.Contains(out, iface)).To(BeFalse())
}

func forwardPort(k8s *kube_testing.K8s, pod *v1.Pod, port int) *kube_testing.PortForward {
	forwarding, err := k8s.NewPortForwarder(pod, port)
	Expect(err).To(BeNil())

	err = forwarding.Start()
	Expect(err).To(BeNil())
	logrus.Infof("Forwarded port: pod=%s, remote=%d local=%d\n", pod.Name, port, forwarding.ListenPort)
	return forwarding
}

func connectDataplane(port int) dataplaneapi.DataplaneClient {
	dataplaneConn, err := tools.SocketOperationCheck(localPort(dataplaneSocketType, port))
	Expect(err).To(BeNil())
	return dataplaneapi.NewDataplaneClient(dataplaneConn)
}

func unmaskIp(maskedIp string) string {
	return strings.Split(maskedIp, "/")[0]
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

func dataplanePodTemplate(node *v1.Node) *v1.Pod {
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
		},
		v1.ContainerPort{
			ContainerPort: 40001,
			Name:          "debug-agent",
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
