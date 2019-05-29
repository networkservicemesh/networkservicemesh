package model

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"sync"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
)

// DataplaneState describes state of dataplane
type DataplaneState int8

const (
	// DataplaneStateNone means there is no active connection in dataplane
	DataplaneStateNone DataplaneState = 0 // In case dataplane is not yet configured for connection

	// DataplaneStateReady means there is an active connection in dataplane
	DataplaneStateReady DataplaneState = 1 // In case dataplane is configured for connection.
)

// Dataplane structure in Model that describes dataplane
type Dataplane struct {
	RegisteredName       string
	SocketLocation       string
	LocalMechanisms      []*local.Mechanism
	RemoteMechanisms     []*remote.Mechanism
	MechanismsConfigured bool
}

// Clone returns pointer to copy of Dataplane
func (d *Dataplane) clone() cloneable {
	if d == nil {
		return nil
	}

	lm := make([]*local.Mechanism, 0, len(d.LocalMechanisms))
	for _, m := range d.LocalMechanisms {
		lm = append(lm, proto.Clone(m).(*local.Mechanism))
	}

	rm := make([]*remote.Mechanism, 0, len(d.RemoteMechanisms))
	for _, m := range d.RemoteMechanisms {
		rm = append(rm, proto.Clone(m).(*remote.Mechanism))
	}

	return &Dataplane{
		RegisteredName:       d.RegisteredName,
		SocketLocation:       d.SocketLocation,
		LocalMechanisms:      lm,
		RemoteMechanisms:     rm,
		MechanismsConfigured: d.MechanismsConfigured,
	}
}

type dataplaneDomain struct {
	baseDomain
	inner sync.Map
}

func newDataplaneDomain() dataplaneDomain {
	return dataplaneDomain{
		baseDomain: newBase(),
	}
}

func (d *dataplaneDomain) AddDataplane(dp *Dataplane) {
	d.store(dp.RegisteredName, dp)
}

func (d *dataplaneDomain) GetDataplane(name string) *Dataplane {
	v, _ := d.load(name)
	if v != nil {
		return v.(*Dataplane)
	}
	return nil
}

func (d *dataplaneDomain) DeleteDataplane(name string) {
	d.delete(name)
}

func (d *dataplaneDomain) UpdateDataplane(dp *Dataplane) {
	d.store(dp.RegisteredName, dp)
}

func (d *dataplaneDomain) SelectDataplane(dataplaneSelector func(dp *Dataplane) bool) (*Dataplane, error) {
	var rv *Dataplane
	d.kvRange(func(key string, value interface{}) bool {
		dp := value.(*Dataplane)

		if dataplaneSelector == nil {
			rv = dp
			return false
		}

		if dataplaneSelector(dp) {
			rv = dp
			return false
		}

		return true
	})

	if rv == nil {
		return nil, fmt.Errorf("no appropriate dataplanes found")
	}

	return rv, nil
}

func (d *dataplaneDomain) SetDataplaneModificationHandler(h *ModificationHandler) func() {
	return d.addHandler(h)
}
