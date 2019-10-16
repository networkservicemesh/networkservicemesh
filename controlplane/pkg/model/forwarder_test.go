package model

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
)

func TestAddAndGetDp(t *testing.T) {
	g := NewWithT(t)

	dp := &Forwarder{
		RegisteredName: "dp1",
		SocketLocation: "/socket",
		LocalMechanisms: []connection.Mechanism{
			&local.Mechanism{
				Type: local.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					"localParam": "value",
				},
			},
		},
		RemoteMechanisms: []connection.Mechanism{
			&remote.Mechanism{
				Type: remote.MechanismType_GRE,
				Parameters: map[string]string{
					"remoteParam": "value",
				},
			},
		},
		MechanismsConfigured: true,
	}

	dd := newForwarderDomain()
	dd.AddForwarder(context.Background(), dp)
	getDp := dd.GetForwarder("dp1")

	g.Expect(getDp.RegisteredName).To(Equal(dp.RegisteredName))
	g.Expect(getDp.SocketLocation).To(Equal(dp.SocketLocation))
	g.Expect(getDp.MechanismsConfigured).To(Equal(dp.MechanismsConfigured))
	g.Expect(getDp.LocalMechanisms).To(Equal(dp.LocalMechanisms))
	g.Expect(getDp.RemoteMechanisms).To(Equal(dp.RemoteMechanisms))

	g.Expect(fmt.Sprintf("%p", getDp.LocalMechanisms)).ToNot(Equal(fmt.Sprintf("%p", dp.LocalMechanisms)))
	g.Expect(fmt.Sprintf("%p", getDp.RemoteMechanisms)).ToNot(Equal(fmt.Sprintf("%p", dp.RemoteMechanisms)))
}

func TestDeleteDp(t *testing.T) {
	g := NewWithT(t)

	dd := newForwarderDomain()
	dd.AddForwarder(context.Background(), &Forwarder{
		RegisteredName: "dp1",
		SocketLocation: "/socket",
		LocalMechanisms: []connection.Mechanism{
			&local.Mechanism{
				Type: local.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					"localParam": "value",
				},
			},
		},
		RemoteMechanisms: []connection.Mechanism{
			&remote.Mechanism{
				Type: remote.MechanismType_GRE,
				Parameters: map[string]string{
					"remoteParam": "value",
				},
			},
		},
		MechanismsConfigured: true,
	})

	cc := dd.GetForwarder("dp1")
	g.Expect(cc).ToNot(BeNil())

	dd.DeleteForwarder(context.Background(), "dp1")

	dpDel := dd.GetForwarder("dp1")
	g.Expect(dpDel).To(BeNil())

	dd.DeleteForwarder(context.Background(), "NotExistingId")
}

func TestSelectDp(t *testing.T) {
	g := NewWithT(t)

	amount := 5
	dd := newForwarderDomain()
	for i := 0; i < amount; i++ {
		dd.AddForwarder(context.Background(), &Forwarder{
			RegisteredName: fmt.Sprintf("dp%d", i),
			SocketLocation: fmt.Sprintf("/socket-%d", i),
			LocalMechanisms: []connection.Mechanism{
				&local.Mechanism{
					Type: local.MechanismType_MEM_INTERFACE,
					Parameters: map[string]string{
						"localParam": "value",
					},
				},
			},
			RemoteMechanisms: []connection.Mechanism{
				&remote.Mechanism{
					Type: remote.MechanismType_GRE,
					Parameters: map[string]string{
						"remoteParam": "value",
					},
				},
			},
			MechanismsConfigured: true,
		})
	}

	selector := func(dp *Forwarder) bool {
		return dp.SocketLocation == "/socket-4"
	}

	selectedDp, err := dd.SelectForwarder(selector)
	g.Expect(err).To(BeNil())
	g.Expect(selectedDp.RegisteredName).To(Equal("dp4"))

	emptySelector := func(dp *Forwarder) bool {
		return false
	}
	selectedDp, err = dd.SelectForwarder(emptySelector)
	g.Expect(err.Error()).To(ContainSubstring("no appropriate forwarders found"))
	g.Expect(selectedDp).To(BeNil())

	first, err := dd.SelectForwarder(nil)
	g.Expect(err).To(BeNil())
	g.Expect(first.RegisteredName).ToNot(BeNil())
}
