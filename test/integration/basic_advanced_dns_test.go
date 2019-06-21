// +build basic

package nsmd_integration_tests

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
)

func TestAdvancedDNS(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	coreFile1 := `.:53 {
    log
    hosts {
        172.16.1.2 trash.com
        fallthrough
    }
}`
	coreFile2 := `.:53 {
    log
    hosts {
        9.9.9.9 trash2.com
        fallthrough
    }
	forward . 172.16.1.1
}`
	k8s, err := kubetest.NewK8s(true)
	Expect(err).Should(BeNil())
	defer k8s.Cleanup()
	createDnsConfig(k8s, "core1", coreFile1)
	createDnsConfig(k8s, "core2", coreFile2)

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.FailLogger(k8s, nodes, t)

	kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	nscAndDns1 := kubetest.DeployNSCDns(k8s, nodes[0].Node, "nsc1", "core1", defaultTimeout)
	nscAndDns2 := kubetest.DeployNSCDns(k8s, nodes[0].Node, "nsc2", "core2", defaultTimeout)
	Expect(true, kubetest.IsNsePinged(k8s, nscAndDns1))
	Expect(true, kubetest.IsNsePinged(k8s, nscAndDns2))
}

func createDnsConfig(k8s *kubetest.K8s,name, content string){
	_, err := k8s.CreateConfigMap(&v1.ConfigMap{
		TypeMeta : metav1.TypeMeta{
			Kind:"ConfigMap",
			APIVersion:"v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:name,
			Namespace:k8s.GetK8sNamespace(),
		},

		BinaryData: map[string][]byte{
			"Corefile": []byte(content),
		},
	})

	Expect(err).To(BeNil())

}
