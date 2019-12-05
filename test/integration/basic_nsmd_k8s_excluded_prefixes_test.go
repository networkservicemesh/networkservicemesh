// +build basic

package nsmd_integration_tests

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta2"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestK8sExcludedPrefixes(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	clientset, err := k8s.GetClientSet()
	g.Expect(err).To(BeNil())
	cm, err := clientset.CoreV1().ConfigMaps("kube-system").Get("kubeadm-config", metav1.GetOptions{})

	if cm == nil || err != nil {
		t.Skip("Skip, no kubeadm-config")
		return
	}

	clusterConfiguration := &v1beta2.ClusterConfiguration{}
	err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(cm.Data["ClusterConfiguration"]), 4096).
		Decode(clusterConfiguration)
	g.Expect(err).To(BeNil())

	podSubnet := clusterConfiguration.Networking.PodSubnet
	serviceSubnet := clusterConfiguration.Networking.ServiceSubnet

	pattern := "context:<ip_context:<src_ip_required:true dst_ip_required:true excluded_prefixes:\\\"" + podSubnet + "\\\" excluded_prefixes:\\\"" + serviceSubnet + "\\\" > >"

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	defer kubetest.MakeLogsSnapshot(k8s, t)

	icmp := kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder-nse-1", defaultTimeout)

	nsc, err := clientset.CoreV1().Pods(k8s.GetK8sNamespace()).Create(pods.NSCPod("nsc", nodes[0].Node,
		map[string]string{
			"OUTGOING_NSC_LABELS":    "app=icmp",
			"CLIENT_NETWORK_SERVICE": "icmp-responder",
		},
	))

	defer k8s.DeletePods(nsc)

	g.Expect(err).To(BeNil())
	k8s.WaitLogsContains(icmp, "", pattern, defaultTimeout)
}
