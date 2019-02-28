package nsmd_integration_tests

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"strings"
	"testing"
	"time"
)

const (
	vppContainer = ""
)

func TestVppPostmortemDataCollection(t *testing.T) {
	RegisterTestingT(t)

	k8s, dataplane := deployVppDataplane(defaultTimeout)

	defer k8s.Cleanup()
	defer clearPostmortemData(k8s, dataplane) // prevent false alarms

	vpp := getVppPID(k8s, dataplane)
	sendSignal(k8s, dataplane, vpp, "SIGSEGV")

	k8s.WaitLogsContains(dataplane, vppContainer, "Postmortem data collection finished", defaultTimeout)

	backtraces := postmortemFiles(k8s, dataplane)
	Expect(backtraces).NotTo(BeEmpty())
}

func deployVppDataplane(timeout time.Duration) (*kube_testing.K8s, *v1.Pod) {
	k8s, err := kube_testing.NewK8s()
	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.Prepare("nsmd", "nsc", "nsmd-dataplane", "icmp-responder-nse", "jaeger")
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	// prepare node
	nodes := k8s.GetNodesWait(1, timeout)
	Expect(len(nodes) >= 1).To(Equal(true), "At least one kubernetes node is required for this test")

	// deploy vpp-dataplane pod
	node := &nodes[0]
	dataplaneName := fmt.Sprintf("nsmd-dataplane-%s", node.Name)
	dataplane := k8s.CreatePod(pods.VPPDataplanePod(dataplaneName, node))

	// wait gdb monitor is initialized and attached to the vpp
	k8s.WaitLogsContains(dataplane, "", "GDB Monitor attached successfully", timeout)

	return k8s, dataplane
}

func getVppPID(k8s *kube_testing.K8s, pod *v1.Pod) string {
	getVppPidCommand := []string{"supervisorctl", "-c", "/etc/supervisord/supervisord.conf", "pid", "vpp"}

	stdout, stderr, err := k8s.Exec(pod, vppContainer, getVppPidCommand...)
	Expect(err).To(BeNil())
	Expect(stderr).To(BeEmpty())
	return strings.Trim(stdout, " \t\n")
}

func sendSignal(k8s *kube_testing.K8s, pod *v1.Pod, pid, signal string) {
	command := []string{"kill", "-s", signal, pid}

	_, stderr, err := k8s.Exec(pod, vppContainer, command...)
	Expect(err).To(BeNil())
	Expect(stderr).To(BeEmpty())
}

func postmortemFiles(k8s *kube_testing.K8s, dataplane *v1.Pod) []string {
	return k8s.FindFiles(dataplane, vppContainer, pods.VppPostmortemDataLocation, pods.VppBacktracePattern)
}

func clearPostmortemData(k8s *kube_testing.K8s, dataplane *v1.Pod) {
	_, _, _ = k8s.Exec(dataplane, vppContainer, "rm", "-rf", pods.VppPostmortemDataLocation)
}
