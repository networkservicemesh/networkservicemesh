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

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.PrepareDefault()
	st := time.Now()
	nodes := nsmd_test_utils.SetupNodes(k8s, 1, defaultTimeout)
	k8s.DeletePods(nodes[0].Dataplane)
	k8s.Cleanup()
	logrus.Infof("Deploy/Shutdown time: %v", time.Since(st))
}
