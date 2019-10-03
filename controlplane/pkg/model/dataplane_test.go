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

	dp := &Dataplane{
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

	dd := newDataplaneDomain()
	dd.AddDataplane(context.Background(), dp)
	getDp := dd.GetDataplane("dp1")

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

	dd := newDataplaneDomain()
	dd.AddDataplane(context.Background(), &Dataplane{
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

	cc := dd.GetDataplane("dp1")
	g.Expect(cc).ToNot(BeNil())

	dd.DeleteDataplane(context.Background(), "dp1")

	dpDel := dd.GetDataplane("dp1")
	g.Expect(dpDel).To(BeNil())

	dd.DeleteDataplane(context.Background(), "NotExistingId")
}

func TestSelectDp(t *testing.T) {
	g := NewWithT(t)

	amount := 5
	dd := newDataplaneDomain()
	for i := 0; i < amount; i++ {
		dd.AddDataplane(context.Background(), &Dataplane{
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

	selector := func(dp *Dataplane) bool {
		return dp.SocketLocation == "/socket-4"
	}

	selectedDp, err := dd.SelectDataplane(selector)
	g.Expect(err).To(BeNil())
	g.Expect(selectedDp.RegisteredName).To(Equal("dp4"))

	emptySelector := func(dp *Dataplane) bool {
		return false
	}
	selectedDp, err = dd.SelectDataplane(emptySelector)
	g.Expect(err.Error()).To(ContainSubstring("no appropriate dataplanes found"))
	g.Expect(selectedDp).To(BeNil())

	first, err := dd.SelectDataplane(nil)
	g.Expect(err).To(BeNil())
	g.Expect(first.RegisteredName).ToNot(BeNil())
}
