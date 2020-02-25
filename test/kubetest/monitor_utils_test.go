package kubetest

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
)

func newXcon(eventType crossconnect.CrossConnectEventType, srcUp, dstUp, empty bool) *crossconnect.CrossConnectEvent {
	srcState := networkservice.State_DOWN
	if srcUp {
		srcState = networkservice.State_UP
	}

	dstState := networkservice.State_DOWN
	if dstUp {
		dstState = networkservice.State_UP
	}

	xcons := map[string]*crossconnect.CrossConnect{}
	if !empty {
		xcons = map[string]*crossconnect.CrossConnect{
			"newid": {
				Id: "newid",
				Source: &networkservice.Connection{
					Id:    "1",
					State: srcState,
					Path:  common.Strings2Path("local"),
				},
				Destination: &networkservice.Connection{
					Id:    "2",
					State: dstState,
					Path:  common.Strings2Path("local"),
				},
			},
		}
	}

	return &crossconnect.CrossConnectEvent{
		Type:          eventType,
		CrossConnects: xcons,
	}
}

func TestSingleEventChecker(t *testing.T) {
	g := NewWithT(t)

	ch := make(chan *crossconnect.CrossConnectEvent)
	ec := &SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	}

	go func() {
		ch <- newXcon(crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, true, true, false)
	}()

	g.Expect(ec.Check(ch)).To(BeNil())
}

func TestMultipleEventChecker(t *testing.T) {
	g := NewWithT(t)

	ch := make(chan *crossconnect.CrossConnectEvent)
	ec := &MultipleEventChecker{
		Events: []EventChecker{
			&SingleEventChecker{
				EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
				Empty:     true,
			},
			&SingleEventChecker{
				EventType: crossconnect.CrossConnectEventType_UPDATE,
				SrcUp:     true,
				DstUp:     true,
			},
			&SingleEventChecker{
				EventType: crossconnect.CrossConnectEventType_DELETE,
			},
		},
	}

	go func() {
		ch <- newXcon(crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, false, false, true)
		ch <- newXcon(crossconnect.CrossConnectEventType_UPDATE, true, true, false)
		ch <- newXcon(crossconnect.CrossConnectEventType_DELETE, false, false, false)
	}()

	g.Expect(ec.Check(ch)).To(BeNil())
}

func TestOrEventChecker_FirstSuccess(t *testing.T) {
	g := NewWithT(t)

	ch := make(chan *crossconnect.CrossConnectEvent)
	ec := &OrEventChecker{
		Event1: &MultipleEventChecker{
			Events: []EventChecker{
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
					Empty:     true,
				},
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_UPDATE,
					SrcUp:     true,
					DstUp:     true,
				},
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_DELETE,
				},
			},
		},
		Event2: &MultipleEventChecker{
			Events: []EventChecker{
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
					SrcUp:     true,
					DstUp:     true,
				},
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_UPDATE,
					SrcUp:     true,
					DstUp:     true,
				},
			},
		},
	}

	go func() {
		ch <- newXcon(crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, false, false, true)
		ch <- newXcon(crossconnect.CrossConnectEventType_UPDATE, true, true, false)
		ch <- newXcon(crossconnect.CrossConnectEventType_DELETE, false, false, false)
	}()

	g.Expect(ec.Check(ch)).To(BeNil())
}

func TestOrEventChecker_SecondSuccess(t *testing.T) {
	g := NewWithT(t)

	ch := make(chan *crossconnect.CrossConnectEvent)
	ec := &OrEventChecker{
		Event1: &MultipleEventChecker{
			Events: []EventChecker{
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
					Empty:     true,
				},
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_UPDATE,
					SrcUp:     true,
					DstUp:     true,
				},
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_DELETE,
				},
			},
		},
		Event2: &MultipleEventChecker{
			Events: []EventChecker{
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
					SrcUp:     true,
					DstUp:     true,
				},
				&SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_UPDATE,
					SrcUp:     true,
					DstUp:     true,
				},
			},
		},
	}

	go func() {
		ch <- newXcon(crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, true, true, false)
		ch <- newXcon(crossconnect.CrossConnectEventType_UPDATE, true, true, false)
	}()

	g.Expect(ec.Check(ch)).To(BeNil())
}
