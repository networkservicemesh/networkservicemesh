// +build recover

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
	"testing"
)

func TestXconMonitorSingleNodeHealFailed(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())

	icmpPod := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-0", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nscPodNode).ToNot(BeNil())

	eventCh, closeFunc := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFunc()

	expectedCh := make(chan kubetest.EventDescription, 10)
	waitCh := make(chan struct{})

	go kubetest.CheckEventsCh(t, eventCh, expectedCh, waitCh)

	k8s.DeletePods(icmpPod)

	expectedCh <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	}

	expectedCh <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	}

	expectedCh <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		LastEvent: true,
	}

	<-waitCh
	close(expectedCh)
}

func TestXconMonitorSingleNodeHealSuccess(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())

	icmp0 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-0", defaultTimeout)
	Expect(icmp0).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nsc).ToNot(BeNil())

	eventCh, closeFunc := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFunc()

	expectedCh := make(chan kubetest.EventDescription, 10)
	waitCh := make(chan struct{})

	go kubetest.CheckEventsCh(t, eventCh, expectedCh, waitCh)

	icmp1 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-1", defaultTimeout)
	Expect(icmp1).ToNot(BeNil())

	k8s.DeletePods(icmp0)

	expectedCh <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	}

	expectedCh <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	}

	expectedCh <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     true,
		LastEvent: true,
	}

	<-waitCh
	close(expectedCh)
}

func TestXconMonitorMultiNodeHealFail(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	icmp := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-0", defaultTimeout)
	Expect(icmp).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nsc).ToNot(BeNil())

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.CrossConnectClientAt(k8s, nodesConf[1].Nsmd)
	defer closeFunc1()

	// checking goroutine for node0
	expectedCh0 := make(chan kubetest.EventDescription, 10)
	waitCh0 := make(chan struct{})

	go kubetest.CheckEventsCh(t, eventCh0, expectedCh0, waitCh0)

	// checking goroutine for node1
	expectedCh1 := make(chan kubetest.EventDescription, 10)
	waitCh1 := make(chan struct{})

	go kubetest.CheckEventsCh(t, eventCh1, expectedCh1, waitCh1)

	k8s.DeletePods(icmp)

	// expected events for node0
	expectedCh0 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	}

	expectedCh0 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	}

	expectedCh0 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		LastEvent: true,
	}

	// expected events for node1
	expectedCh1 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	}

	expectedCh1 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	}

	expectedCh1 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		LastEvent: true,
	}

	<-waitCh0
	close(expectedCh0)

	<-waitCh1
	close(expectedCh1)
}

func TestXconMonitorMultiNodeHealSuccess(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	icmp0 := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-0", defaultTimeout)
	Expect(icmp0).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nsc).ToNot(BeNil())

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.CrossConnectClientAt(k8s, nodesConf[1].Nsmd)
	defer closeFunc1()

	// checking goroutine for node0
	expectedCh0 := make(chan kubetest.EventDescription, 10)
	waitCh0 := make(chan struct{})

	go kubetest.CheckEventsCh(t, eventCh0, expectedCh0, waitCh0)

	// checking goroutine for node1
	expectedCh1 := make(chan kubetest.EventDescription, 10)
	waitCh1 := make(chan struct{})

	go kubetest.CheckEventsCh(t, eventCh1, expectedCh1, waitCh1)

	icmp1 := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-1", defaultTimeout)
	Expect(icmp1).ToNot(BeNil())
	k8s.DeletePods(icmp0)

	// expected events for node0
	expectedCh0 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	}

	expectedCh0 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	}

	expectedCh0 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     true,
		LastEvent: true,
	}

	// expected events for node1
	expectedCh1 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	}

	expectedCh1 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	}

	expectedCh1 <- kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		LastEvent: true,
	}

	<-waitCh0
	close(expectedCh0)

	<-waitCh1
	close(expectedCh1)
}
