package endpoint

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
)

//CreateRouteMutator - Creates an instance of ConnectionMutator with routes mutating
func CreateRouteMutator(routes []string) ConnectionMutator {
	return func(c *connection.Connection) error {
		for _, r := range routes {
			c.Context.IpContext.SrcRoutes = append(c.Context.IpContext.SrcRoutes, &connectioncontext.Route{
				Prefix: r,
			})
		}
		return nil
	}
}
