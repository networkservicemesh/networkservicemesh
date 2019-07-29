package nsmd_integration_tests

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func TestCertSidecar(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	g.Expect(err).To(BeNil())

	checkSpire(k8s)

	kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-responder-nse-1", defaultTimeout)
	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-1", defaultTimeout)

	checkSpire(k8s)

	logs, err := k8s.GetLogs(nsc, "nsm-init")
	g.Expect(err).To(BeNil())
	logrus.Infof(logs)
}

func checkSpire(k8s *kubetest.K8s) {
	cs, err := k8s.GetClientSet()
	Expect(err).To(BeNil())

	pl, err := cs.CoreV1().Pods("spire").List(v1.ListOptions{})
	Expect(err).To(BeNil())

	//for _, p := range pl.Items {
	for i := 0; i < len(pl.Items); i++ {
		logrus.Infof("====== %v ======", pl.Items[i].Name)
		raw, err := cs.CoreV1().Pods("spire").GetLogs(pl.Items[i].Name, &v12.PodLogOptions{}).DoRaw()
		Expect(err).To(BeNil())
		logrus.Info(string(raw))
		logrus.Info("================", pl.Items[i].Name)
	}
}
