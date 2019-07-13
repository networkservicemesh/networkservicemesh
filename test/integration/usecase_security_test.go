// +build security

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCertSidecar(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	checkSpire(k8s)

	kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-1", defaultTimeout)

	checkSpire(k8s)

	logs, err := k8s.GetLogs(nsc, "nsm-init")
	logrus.Infof(logs)
	return
}

func checkSpire(k8s *kubetest.K8s) {
	cs, err := k8s.GetClientSet()
	Expect(err).To(BeNil())

	pl, err := cs.CoreV1().Pods("spire").List(v1.ListOptions{})
	Expect(err).To(BeNil())

	for _, p := range pl.Items {
		logrus.Infof("====== %v ======", p.Name)
		raw, err := cs.CoreV1().Pods("spire").GetLogs(p.Name, &v12.PodLogOptions{}).DoRaw()
		Expect(err).To(BeNil())
		logrus.Info(string(raw))
		logrus.Info("================", p.Name)
	}
}
