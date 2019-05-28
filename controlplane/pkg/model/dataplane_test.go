package model

import (
	"fmt"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAddAndGetDp(t *testing.T) {
	RegisterTestingT(t)

	dp := &Dataplane{
		RegisteredName: "dp1",
		SocketLocation: "/socket",
		LocalMechanisms: []*local.Mechanism{
			{
				Type: local.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					"localParam": "value",
				},
			},
		},
		RemoteMechanisms: []*remote.Mechanism{
			{
				Type: remote.MechanismType_GRE,
				Parameters: map[string]string{
					"remoteParam": "value",
				},
			},
		},
		MechanismsConfigured: true,
	}

	dd := dataplaneDomain{}
	dd.AddDataplane(dp)
	getDp := dd.GetDataplane("dp1")

	Expect(getDp.RegisteredName).To(Equal(dp.RegisteredName))
	Expect(getDp.SocketLocation).To(Equal(dp.SocketLocation))
	Expect(getDp.MechanismsConfigured).To(Equal(dp.MechanismsConfigured))
	Expect(getDp.LocalMechanisms).To(Equal(dp.LocalMechanisms))
	Expect(getDp.RemoteMechanisms).To(Equal(dp.RemoteMechanisms))

	Expect(fmt.Sprintf("%p", getDp.LocalMechanisms)).ToNot(Equal(fmt.Sprintf("%p", dp.LocalMechanisms)))
	Expect(fmt.Sprintf("%p", getDp.RemoteMechanisms)).ToNot(Equal(fmt.Sprintf("%p", dp.RemoteMechanisms)))
}

func TestDeleteDp(t *testing.T) {
	RegisterTestingT(t)

	dd := dataplaneDomain{}
	dd.AddDataplane(&Dataplane{
		RegisteredName: "dp1",
		SocketLocation: "/socket",
		LocalMechanisms: []*local.Mechanism{
			{
				Type: local.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					"localParam": "value",
				},
			},
		},
		RemoteMechanisms: []*remote.Mechanism{
			{
				Type: remote.MechanismType_GRE,
				Parameters: map[string]string{
					"remoteParam": "value",
				},
			},
		},
		MechanismsConfigured: true,
	})

	cc := dd.GetDataplane("dp1")
	Expect(cc).ToNot(BeNil())

	dd.DeleteDataplane("dp1")

	dpDel := dd.GetDataplane("dp1")
	Expect(dpDel).To(BeNil())

	dd.DeleteDataplane("NotExistingId")
}

func TestSelectDp(t *testing.T) {
	RegisterTestingT(t)

	amount := 5
	dd := dataplaneDomain{}
	for i := 0; i < amount; i++ {
		dd.AddDataplane(&Dataplane{
			RegisteredName: fmt.Sprintf("dp%d", i),
			SocketLocation: fmt.Sprintf("/socket-%d", i),
			LocalMechanisms: []*local.Mechanism{
				{
					Type: local.MechanismType_MEM_INTERFACE,
					Parameters: map[string]string{
						"localParam": "value",
					},
				},
			},
			RemoteMechanisms: []*remote.Mechanism{
				{
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
	Expect(err).To(BeNil())
	Expect(selectedDp.RegisteredName).To(Equal("dp4"))

	emptySelector := func(dp *Dataplane) bool {
		return false
	}
	selectedDp, err = dd.SelectDataplane(emptySelector)
	Expect(err.Error()).To(ContainSubstring("no appropriate dataplanes found"))
	Expect(selectedDp).To(BeNil())

	first, err := dd.SelectDataplane(nil)
	Expect(err).To(BeNil())
	Expect(first.RegisteredName).ToNot(BeNil())
}
