package kubetest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	defaultEventTimeout = 10 * time.Second
)

// MonitorClient is shorter name for crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient
type MonitorClient crossconnect.MonitorCrossConnect_MonitorCrossConnectsClient

// EventChecker abstracts checker of particular events from channel
type EventChecker interface {
	Check(eventCh <-chan *crossconnect.CrossConnectEvent, name string) error
}

// SingleEventChecker checks single event in channel
type SingleEventChecker struct {
	Timeout   time.Duration
	EventType crossconnect.CrossConnectEventType
	Empty     bool
	SrcUp     bool
	DstUp     bool
}

// Check implements method from EventChecker interface
func (e *SingleEventChecker) Check(eventCh <-chan *crossconnect.CrossConnectEvent, name string) error {
	if e.Timeout == 0 {
		e.Timeout = defaultEventTimeout
	}

	select {
	case actual, ok := <-eventCh:
		if !ok {
			return errors.New("end of channel")
		}
		logrus.Infof("%v: Accept new event, type - %v", name, actual.GetType())

		if actual.GetType() != e.EventType {
			return fmt.Errorf("%v event type %v expected to be %v", name, actual.GetType(), e.EventType)
		}

		if actual.GetType() == crossconnect.CrossConnectEventType_DELETE {
			// we don't care about state of connections since event is DELETE
			return nil
		}

		if e.Empty != (len(actual.GetCrossConnects()) == 0) {
			return fmt.Errorf("%v: expected event with 1 cross-connect, actual - %v", name, len(actual.GetCrossConnects()))
		}

		for _, xcon := range actual.GetCrossConnects() {
			logrus.Infof("%s: %v", name, XconToString(xcon))
			if err := checkXcon(xcon, e.SrcUp, e.DstUp, name); err != nil {
				return err
			}
		}
	case <-time.After(e.Timeout):
		return fmt.Errorf("%v: no event during %v seconds, type %v", name, e.Timeout, e.EventType)
	}

	return nil
}

// MultipleEventChecker checks subsequence of events in channel
type MultipleEventChecker struct {
	Events []EventChecker
}

// Check implements method from EventChecker interface
func (e *MultipleEventChecker) Check(eventCh <-chan *crossconnect.CrossConnectEvent, name string) error {
	if len(e.Events) == 0 {
		return errors.New("events array can't be empty")
	}

	for _, event := range e.Events {
		if err := event.Check(eventCh, name); err != nil {
			return err
		}
	}

	return nil
}

// OrEventChecker checks that one of checker - Event1 or Event2 successfully finishes
type OrEventChecker struct {
	Event1 EventChecker
	Event2 EventChecker
}

// Check implements method from EventChecker interface
func (e *OrEventChecker) Check(eventCh <-chan *crossconnect.CrossConnectEvent, name string) error {
	m := sync.Mutex{}
	copyCh := make(chan *crossconnect.CrossConnectEvent)
	var buffer []*crossconnect.CrossConnectEvent
	go func() {
		for {
			m.Lock()
			event := <-eventCh
			buffer = append(buffer, event)
			m.Unlock()
			copyCh <- event

		}
	}()

	err := e.Event1.Check(copyCh, name)
	if err == nil {
		return nil
	}
	logrus.Infof("%v: the first condition is false: %v, checking the second...", name, err)

	copyCh2 := make(chan *crossconnect.CrossConnectEvent)
	go func() {
		m.Lock()
		for _, event := range buffer {
			copyCh2 <- event
		}
		event := <-eventCh
		m.Unlock()
		copyCh2 <- event
	}()

	if err := e.Event2.Check(copyCh2, name); err != nil {
		return err
	}

	return nil
}

// CrossConnectClientAt returns channel of CrossConnectEvents from passed nsmgr pod
func CrossConnectClientAt(k8s *K8s, pod *v1.Pod, suffix string) (<-chan *crossconnect.CrossConnectEvent, func()) {
	fwd, err := k8s.NewPortForwarder(pod, 5001)
	k8s.g.Expect(err).To(BeNil())

	err = fwd.Start()
	k8s.g.Expect(err).To(BeNil())

	client, closeClient, cancel := CreateCrossConnectClient(k8s, fmt.Sprintf("localhost:%d", fwd.ListenPort))

	stopCh := make(chan struct{})

	closeFunc := func() {
		logrus.Infof("CLOSE")
		close(stopCh)
		closeClient()
		fwd.Stop()
	}

	return getEventCh(client, cancel, stopCh, suffix), closeFunc
}

// NewEventChecker starts goroutine that read events from actualCh and
// compare it with SingleEventChecker passed to expectedFunc
func NewEventChecker(t *testing.T, actualCh <-chan *crossconnect.CrossConnectEvent, name string) (expectedFunc func(EventChecker), waitFunc func()) {
	expectedCh := make(chan EventChecker, 10)
	waitCh := make(chan struct{})

	go checkEventsCh(t, actualCh, expectedCh, waitCh, name)

	expectedFunc = func(d EventChecker) {
		expectedCh <- d
	}

	waitFunc = func() {
		close(expectedCh)
		<-waitCh
	}

	return
}

func checkEventsCh(t *testing.T, actualCh <-chan *crossconnect.CrossConnectEvent,
	expectedCh <-chan EventChecker, waitCh chan struct{}, name string) {
	defer close(waitCh)

	for checker := range expectedCh {
		if err := checker.Check(actualCh, name); err != nil {
			t.Errorf("%s error = %v", name, err)
			return
		}
	}
}

func checkXcon(actual *crossconnect.CrossConnect, srcUp, dstUp bool, name string) error {
	if src := actual.GetLocalSource(); src != nil && (srcUp != (src.GetState().String() == "UP")) {
		return xconStateError(actual, "SRC_UP", srcUp, name)
	}
	if src := actual.GetRemoteSource(); src != nil && (srcUp != (src.GetState().String() == "UP")) {
		return xconStateError(actual, "SRC_UP", srcUp, name)
	}
	if dst := actual.GetLocalDestination(); dst != nil && dstUp != (dst.GetState().String() == "UP") {
		return xconStateError(actual, "DST_UP", dstUp, name)
	}
	if dst := actual.GetRemoteDestination(); dst != nil && dstUp != (dst.GetState().String() == "UP") {
		return xconStateError(actual, "DST_UP", dstUp, name)
	}
	return nil
}

func xconStateError(xcon *crossconnect.CrossConnect, side string, expected bool, name string) error {
	return fmt.Errorf("%s: %s, expected %s state to be %v", name, XconToString(xcon), side, expected)
}

func getEventCh(mc MonitorClient, cf context.CancelFunc, stopCh <-chan struct{}, suffix string) <-chan *crossconnect.CrossConnectEvent {
	eventCh := make(chan *crossconnect.CrossConnectEvent)

	go func() {
		for {
			select {
			case <-mc.Context().Done():
				logrus.Errorf("%s: Context done", suffix)
				close(eventCh)
				return
			default:
				event, err := mc.Recv()
				logrus.Infof("%v: Recv: %v:", suffix, event)
				if err != nil {
					logrus.Errorf("%v: Recv: %v:", suffix, err)
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
func CreateCrossConnectClient(k8s *K8s, address string) (MonitorClient, func(), context.CancelFunc) {
	var err error
	logrus.Infof("Starting CrossConnections Monitor on %s", address)
	conn, err := tools.DialTCP(address)
	if err != nil {
		k8s.g.Expect(err).To(BeNil())
		return nil, nil, nil
	}

	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := monitorClient.MonitorCrossConnects(ctx, &empty.Empty{})
	if err != nil {
		k8s.g.Expect(err).To(BeNil())
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
