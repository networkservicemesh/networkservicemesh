package model

import (
	"context"

	"github.com/pkg/errors"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
)

// DataplaneState describes state of forwarder
type DataplaneState int8

const (
	// DataplaneStateNone means there is no active connection in forwarder
	DataplaneStateNone DataplaneState = 0 // In case forwarder is not yet configured for connection

	// DataplaneStateReady means there is an active connection in forwarder
	DataplaneStateReady DataplaneState = 1 // In case forwarder is configured for connection.
)

// Dataplane structure in Model that describes forwarder
type Dataplane struct {
	RegisteredName       string
	SocketLocation       string
	LocalMechanisms      []connection.Mechanism
	RemoteMechanisms     []connection.Mechanism
	MechanismsConfigured bool
}

// Clone returns pointer to copy of Dataplane
func (d *Dataplane) clone() cloneable {
	if d == nil {
		return nil
	}

	lm := make([]connection.Mechanism, 0, len(d.LocalMechanisms))
	for _, m := range d.LocalMechanisms {
		lm = append(lm, m.Clone())
	}

	rm := make([]connection.Mechanism, 0, len(d.RemoteMechanisms))
	for _, m := range d.RemoteMechanisms {
		rm = append(rm, m.Clone())
	}

	return &Dataplane{
		RegisteredName:       d.RegisteredName,
		SocketLocation:       d.SocketLocation,
		LocalMechanisms:      lm,
		RemoteMechanisms:     rm,
		MechanismsConfigured: d.MechanismsConfigured,
	}
}

// SetLocalMechanisms sets forwarder local mechanisms
func (d *Dataplane) SetLocalMechanisms(mechanisms []*local.Mechanism) {
	lm := make([]connection.Mechanism, 0, len(mechanisms))
	for _, m := range mechanisms {
		lm = append(lm, m)
	}

	d.LocalMechanisms = lm
}

// SetRemoteMechanisms sets forwarder remote mechanisms
func (d *Dataplane) SetRemoteMechanisms(mechanisms []*remote.Mechanism) {
	rm := make([]connection.Mechanism, 0, len(mechanisms))
	for _, m := range mechanisms {
		rm = append(rm, m)
	}

	d.RemoteMechanisms = rm
}

type forwarderDomain struct {
	baseDomain
}

func newDataplaneDomain() forwarderDomain {
	return forwarderDomain{
		baseDomain: newBase(),
	}
}

func (d *forwarderDomain) AddDataplane(ctx context.Context, dp *Dataplane) {
	d.store(ctx, dp.RegisteredName, dp)
}

func (d *forwarderDomain) GetDataplane(name string) *Dataplane {
	v, _ := d.load(name)
	if v != nil {
		return v.(*Dataplane)
	}
	return nil
}

func (d *forwarderDomain) DeleteDataplane(ctx context.Context, name string) {
	d.delete(ctx, name)
}

func (d *forwarderDomain) UpdateDataplane(ctx context.Context, dp *Dataplane) {
	d.store(ctx, dp.RegisteredName, dp)
}

func (d *forwarderDomain) SelectDataplane(forwarderSelector func(dp *Dataplane) bool) (*Dataplane, error) {
	var rv *Dataplane
	d.kvRange(func(key string, value interface{}) bool {
		dp := value.(*Dataplane)

		if forwarderSelector == nil {
			rv = dp
			return false
		}

		if forwarderSelector(dp) {
			rv = dp
			return false
		}

		return true
	})

	if rv == nil {
		return nil, errors.New("no appropriate forwarders found")
	}

	return rv, nil
}

func (d *forwarderDomain) SetDataplaneModificationHandler(h *ModificationHandler) func() {
	return d.addHandler(h)
}
