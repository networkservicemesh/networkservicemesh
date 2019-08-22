package endpoint

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
)

//CreateRouteMutator - Creates an instance of ConnectionMutator with routes mutating
func CreateRouteMutator(routes []string) ConnectionMutator {
	return func(c *connection.Connection) error {
		for _, r := range routes {
			c.GetContext().GetIpContext().DstRoutes = append(c.GetContext().GetIpContext().GetDstRoutes(), &connectioncontext.Route{
				Prefix: r,
			})
		}
		return nil
	}
}
