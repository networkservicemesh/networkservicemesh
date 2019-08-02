package kubetest

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
)

func newXcon(id string, eventType crossconnect.CrossConnectEventType, srcUp, dstUp, empty bool) *crossconnect.CrossConnectEvent {
	srcState := connection.State_DOWN
	if srcUp {
		srcState = connection.State_UP
	}

	dstState := connection.State_DOWN
	if dstUp {
		dstState = connection.State_UP
	}

	xcons := map[string]*crossconnect.CrossConnect{}
	if !empty {
		xcons = map[string]*crossconnect.CrossConnect{
			id: {
				Id: id,
				Source: &crossconnect.CrossConnect_LocalSource{
					LocalSource: &connection.Connection{
						Id:    "1",
						State: srcState,
					},
				},
				Destination: &crossconnect.CrossConnect_LocalDestination{
					LocalDestination: &connection.Connection{
						Id:    "2",
						State: dstState,
					},
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
		ch <- newXcon("1", crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, true, true, false)
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
		ch <- newXcon("1", crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, false, false, true)
		ch <- newXcon("1", crossconnect.CrossConnectEventType_UPDATE, true, true, false)
		ch <- newXcon("1", crossconnect.CrossConnectEventType_DELETE, false, false, false)
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
		ch <- newXcon("1", crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, false, false, true)
		ch <- newXcon("1", crossconnect.CrossConnectEventType_UPDATE, true, true, false)
		ch <- newXcon("1", crossconnect.CrossConnectEventType_DELETE, false, false, false)
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
		ch <- newXcon("1", crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, true, true, false)
		ch <- newXcon("1", crossconnect.CrossConnectEventType_UPDATE, true, true, false)
	}()

	g.Expect(ec.Check(ch)).To(BeNil())
}
