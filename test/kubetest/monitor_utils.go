package kubetest

import (
	"context"
	"errors"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/api/core/v1"
)

// MonitorClient is shorter name for crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient
type MonitorClient crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient

// EventDescription describes expected event
type EventDescription struct {
	TillNext  time.Duration
	EventType crossconnect.CrossConnectEventType
	SrcUp     bool
	DstUp     bool
	LastEvent bool
}

// CrossConnectClientAt returns channel of CrossConnectEvents from passed nsmgr pod
func CrossConnectClientAt(k8s *K8s, pod *v1.Pod) (<-chan *crossconnect.CrossConnectEvent, func()) {
	fwd, err := k8s.NewPortForwarder(pod, 6001)
	Expect(err).To(BeNil())

	err = fwd.Start()
	Expect(err).To(BeNil())

	client, closeClient, cancel := CreateCrossConnectClient(fmt.Sprintf("localhost:%d", fwd.ListenPort))

	stopCh := make(chan struct{})

	closeFunc := func() {
		logrus.Infof("CLOSE")
		close(stopCh)
		closeClient()
		fwd.Stop()
	}

	return getEventCh(client, cancel, stopCh), closeFunc
}

// XconProxyMonitor deploys proxy monitor to node and returns channel of events from it
func XconProxyMonitor(k8s *K8s, conf *NodeConf, suffix string) (<-chan *crossconnect.CrossConnectEvent, func()) {
	address := fmt.Sprintf("%s:5001", conf.Nsmd.Status.PodIP)

	xconProxy := k8s.CreatePod(pods.TestCommonPod(
		fmt.Sprintf("xcon-proxy-monitor-%s", suffix),
		[]string{"/bin/proxy-xcon-monitor", fmt.Sprintf("-address=%s", address)},
		conf.Node,
		map[string]string{}))

	eventCh, closeFunc := CrossConnectClientAt(k8s, xconProxy)
	return eventCh, func() {
		k8s.DeletePods(xconProxy)
		closeFunc()
	}
}

const defaultNextTimeout = 10 * time.Second

// NewEventChecker starts goroutine that read events from actualCh and
// compare it with EventDescription passed to expectedFunc
func NewEventChecker(t *testing.T, actualCh <-chan *crossconnect.CrossConnectEvent) (expectedFunc func(EventDescription), waitFunc func()) {
	expectedCh := make(chan EventDescription, 10)
	waitCh := make(chan struct{})

	go checkEventsCh(t, actualCh, expectedCh, waitCh)

	expectedFunc = func(d EventDescription) {
		expectedCh <- d
	}

	waitFunc = func() {
		<-waitCh
		close(expectedCh)
	}

	return
}

func checkEventsCh(t *testing.T, actualCh <-chan *crossconnect.CrossConnectEvent,
	expectedCh <-chan EventDescription, waitCh chan struct{}) {
	defer close(waitCh)
	var nextTimeout time.Duration

	for {
		if nextTimeout == 0 {
			nextTimeout = defaultNextTimeout
		}

		select {
		case e, ok := <-actualCh:
			if !ok {
				return
			}
			logrus.Infof("New %v event:", e.Type)
			for _, v := range e.CrossConnects {
				logrus.Infof(XconToString(v))
			}

			expected := <-expectedCh

			if err := checkSingleXconEvent(e, expected); err != nil {
				t.Error(err)
				return
			}

			if expected.LastEvent {
				return
			}
			nextTimeout = expected.TillNext
		case <-time.After(nextTimeout):
			t.Errorf("No events during %v", nextTimeout)
			return
		}
	}
}

func checkSingleXconEvent(actual *crossconnect.CrossConnectEvent, expected EventDescription) error {
	if actual.GetType() != expected.EventType {
		return fmt.Errorf("event type %v expected to be %v", actual.GetType(), expected.EventType)
	}

	if actual.GetType() == crossconnect.CrossConnectEventType_DELETE {
		// we don't care about state of connections since event is DELETE
		return nil
	}

	if len(actual.GetCrossConnects()) != 1 {
		return fmt.Errorf("expected event with 1 cross-connect, actual - %v", len(actual.GetCrossConnects()))
	}

	for _, xcon := range actual.GetCrossConnects() {
		if err := checkXcon(xcon, expected.SrcUp, expected.DstUp); err != nil {
			return err
		}
	}
	return nil
}

func checkXcon(actual *crossconnect.CrossConnect, srcUp, dstUp bool) error {
	if src := actual.GetLocalSource(); src != nil && (srcUp != (src.GetState().String() == "UP")) {
		return xconStateError(actual, "SRC_UP", srcUp)
	}
	if src := actual.GetRemoteSource(); src != nil && (srcUp != (src.GetState().String() == "UP")) {
		return xconStateError(actual, "SRC_UP", srcUp)
	}
	if dst := actual.GetLocalDestination(); dst != nil && dstUp != (dst.GetState().String() == "UP") {
		return xconStateError(actual, "DST_UP", dstUp)
	}
	if dst := actual.GetRemoteDestination(); dst != nil && dstUp != (dst.GetState().String() == "UP") {
		return xconStateError(actual, "DST_UP", dstUp)
	}
	return nil
}

func xconStateError(xcon *crossconnect.CrossConnect, side string, expected bool) error {
	return fmt.Errorf("%s, expected %s state to be %v", XconToString(xcon), side, expected)
}

func getEventCh(mc MonitorClient, cf context.CancelFunc, stopCh <-chan struct{}) <-chan *crossconnect.CrossConnectEvent {
	eventCh := make(chan *crossconnect.CrossConnectEvent)

	go func() {
		for {
			select {
			case <-mc.Context().Done():
				logrus.Error("Context done")
				close(eventCh)
				return
			default:
				event, err := mc.Recv()
				if err != nil {
					logrus.Errorf("Recv: %v:", err)
					continue
				}
				eventCh <- event
			}
		}
	}()

	go func() {
		<-stopCh
		cf()
	}()

	return eventCh
}

// CreateCrossConnectClient returns CrossConnectMonitorClient to passed address
func CreateCrossConnectClient(address string) (MonitorClient, func(), context.CancelFunc) {
	var err error
	logrus.Infof("Starting CrossConnections Monitor on %s", address)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		Expect(err).To(BeNil())
		return nil, nil, nil
	}

	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := monitorClient.MonitorCrossConnects(ctx, &empty.Empty{})
	if err != nil {
		Expect(err).To(BeNil())
		cancel()
		return nil, nil, nil
	}

	closeFunc := func() {
		if err := conn.Close(); err != nil {
			logrus.Errorf("Closing the stream with: %v", err)
		}
	}
	return stream, closeFunc, cancel
}

// XconToString converts CrossConnect to string
func XconToString(xcon *crossconnect.CrossConnect) string {
	return fmt.Sprintf("%s - %s", srcConnToString(xcon), dstConnToString(xcon))
}

func srcConnToString(xcon *crossconnect.CrossConnect) string {
	var ip string
	var state string
	var distance string

	if ls := xcon.GetLocalSource(); ls != nil {
		ip = ls.GetContext().GetIpContext().GetSrcIpAddr()
		state = ls.GetState().String()
		distance = "local"
	} else {
		ip = xcon.GetRemoteSource().GetContext().GetIpContext().GetSrcIpAddr()
		state = xcon.GetRemoteSource().GetState().String()
		distance = "remote"

	}

	return fmt.Sprintf("[SRC:%s:%s:%s]", distance, ip, state)
}

func dstConnToString(xcon *crossconnect.CrossConnect) string {
	var ip string
	var state string
	var distance string
	var endpoint string

	if ls := xcon.GetLocalDestination(); ls != nil {
		ip = ls.GetContext().GetIpContext().GetDstIpAddr()
		state = ls.GetState().String()
		distance = "local"
		endpoint = ls.GetMechanism().GetParameters()[connection.WorkspaceNSEName]
	} else {
		ip = xcon.GetRemoteDestination().GetContext().GetIpContext().GetDstIpAddr()
		state = xcon.GetRemoteDestination().GetState().String()
		distance = "remote"
		endpoint = xcon.GetRemoteDestination().GetMechanism().GetParameters()[connection.WorkspaceNSEName]

	}

	return fmt.Sprintf("[DST:%s:%s:%s:%s]", endpoint, distance, ip, state)
}

// CollectXcons takes n crossconencts from event channel
func CollectXcons(ch <-chan *crossconnect.CrossConnectEvent,
	n int, timeout time.Duration) (map[string]*crossconnect.CrossConnect, error) {
	rv := map[string]*crossconnect.CrossConnect{}
	for {
		select {
		case event, ok := <-ch:
			if !ok && len(rv) < n {
				return nil, errors.New("reached end of CrossConnectEvent channel")
			}
			for id, xcon := range event.GetCrossConnects() {
				rv[id] = xcon
			}
			if len(rv) == n {
				return rv, nil
			}
		case <-time.After(timeout):
			return nil, fmt.Errorf("no events during %v", timeout)
		}
	}
}
