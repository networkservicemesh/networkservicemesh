// +build basic

package nsmd_integration_tests

import (
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSMDDeploy(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	logrus.Print("Running NSMgr Deploy test")

	k8s, err := kube_testing.NewK8s(true)
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	st := time.Now()
	_ = nsmd_test_utils.SetupNodes(k8s, 1, defaultTimeout)
	deploy := time.Now()
	k8s.Cleanup()
	destroy := time.Now()
	logrus.Infof("Pods Start time: %v", deploy.Sub(st))
	Expect(deploy.Sub(st) < time.Second * 15).To(Equal(true))
	logrus.Infof("Pods Cleanup time: %v", destroy.Sub(deploy))
	Expect(destroy.Sub(deploy) < time.Second * 25).To(Equal(true))
}
