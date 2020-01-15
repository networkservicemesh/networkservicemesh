package acl

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	vppacl "github.com/ligato/vpp-agent/api/models/vpp/acl"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

// ACL is a VPP Agent ACL composite
type acl struct {
	rules []*vppacl.ACL_Rule
}

func NewServer(rules []*vppacl.ACL_Rule) networkservice.NetworkServiceServer {
	return &acl{rules: rules}
}

func (a *acl) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conf := vppagent.Config(ctx)
	a.appendAclConfig(conf)
	return next.Server(ctx).Request(ctx, request)
}

func (a *acl) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	conf := vppagent.Config(ctx)
	a.appendAclConfig(conf)
	return next.Server(ctx).Close(ctx, conn)
}

func (a *acl) appendAclConfig(conf *configurator.Config) {
	if a.rules != nil && len(conf.GetVppConfig().GetInterfaces()) > 0 {
		// TODO - this can likely be changed into just a single ACL, with appending new interface to which it
		// can be applied
		conf.GetVppConfig().Acls = append(conf.GetVppConfig().Acls, &vppacl.ACL{
			Name:  "ingress-acl-" + conf.GetVppConfig().GetInterfaces()[len(conf.GetVppConfig().GetInterfaces())-1].GetName(),
			Rules: a.rules,
			Interfaces: &vppacl.ACL_Interfaces{
				Egress:  []string{},
				Ingress: []string{conf.GetVppConfig().GetInterfaces()[len(conf.GetVppConfig().GetInterfaces())-1].GetName()},
			},
		})
	}
}
