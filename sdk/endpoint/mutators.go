package endpoint

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
)

func CreateRouteMutator(routes []string) ConnectionMutator {
	return func(c *connection.Connection) error {
		for _, r := range routes {
			c.Context.IpContext.Routes = append(c.Context.IpContext.Routes, &connectioncontext.Route{
				Prefix: r,
			})
		}
		return nil
	}
}
